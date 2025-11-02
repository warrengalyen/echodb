package app

import (
	"context"
	"echodb/internal/backup"
	"echodb/internal/command"
	_ "echodb/internal/command/mysql"
	_ "echodb/internal/command/postgres"
	"echodb/internal/config"
	"echodb/internal/connect"
	cmdCfg "echodb/internal/domain/command-config"
	_select "echodb/internal/select"
	t "echodb/internal/term"
	"echodb/pkg/logging"
	"echodb/pkg/utils"
	"fmt"
	"strings"
	"sync"
)

type Env struct {
	ConfigFile string
	DbName     string
	All        bool
	FileLog    string
}

type DBInfo struct {
	Server   config.Server
	Database config.Database
}

type App struct {
	ctx context.Context
	cfg *config.Config
	env *Env
}

func NewApp(ctx context.Context, cfg *config.Config, env *Env) *App {
	return &App{
		ctx: ctx,
		cfg: cfg,
		env: env,
	}
}

func (a *App) MustRun() error {
	if err := a.Run(); err != nil {
		logging.L(a.ctx).Error("failed to run app")
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (a *App) Run() error {
	if a.env.All == false && a.env.DbName != "" {
		logging.L(a.ctx).Info("Running the app with the parameters specified (db list)")
		return a.RunDumpDB()
	}

	if a.env.All == true && a.env.DbName == "" {
		logging.L(a.ctx).Info("Running the app with the parameters specified (db all)")
		return a.RunDumpDB()
	}

	logging.L(a.ctx).Info("Running the app in manual mode with db selection")
	return a.RunDumpManual()
}

func (a *App) RunDumpManual() error {
	logging.L(a.ctx).Info("Prepare server list")

	m := t.New()
	serverList, serverKeys := _select.SelectOptionList(a.cfg.Servers, "")
	m.SetList(serverKeys)
	m.SetTitle("Select server")

	if err := runWithCtx(a.ctx, func() error { m.Run(); return nil }); err != nil {
		return err
	}

	serverName := m.GetSelect()
	serverKey := serverList[serverName]
	server := a.cfg.Servers[serverKey]

	logging.L(a.ctx).Info("Selected server", logging.StringAttr("server", serverKey))

	m.ClearList()
	logging.L(a.ctx).Info("Prepare database list")

	dbList, dbKeys := _select.SelectOptionList(a.cfg.Databases, serverKey)
	m.SetList(dbKeys)
	m.SetTitle("Select database")

	if err := runWithCtx(a.ctx, func() error { m.Run(); return nil }); err != nil {
		return err
	}

	dbName := m.GetSelect()
	dbKey := dbList[dbName]
	db := a.cfg.Databases[dbKey]

	logging.L(a.ctx).Info("Selected database", logging.StringAttr("database", dbKey))

	if err := a.runBackup(server, db); err != nil {
		return err
	}

	return nil
}

func (a *App) RunDumpDB() error {
	logging.L(a.ctx).Info("Prepare data for creating dumps")

	dbList := strings.Split(a.env.DbName, ",")
	countDBs := len(dbList)

	serversDatabases := make(map[string][]DBInfo)

	for _, dbName := range dbList {
		database, ok := a.cfg.Databases[dbName]
		if !ok {
			fmt.Printf("Database %s not found\n", dbName)
			logging.L(a.ctx).Warn("Database not found", logging.StringAttr("name", dbName))
			countDBs--
			continue
		}

		server, ok := a.cfg.Servers[database.Server]
		if !ok {
			fmt.Printf("Server %s not found\n", database.Server)
			logging.L(a.ctx).Warn("Server not found", logging.StringAttr("name", database.Server))
			countDBs--
			continue
		}

		serversDatabases[database.Server] = append(serversDatabases[database.Server], DBInfo{
			Server:   server,
			Database: database,
		})

		if countDBs == 0 {
			logging.L(a.ctx).Error("Database and server keys do not match, check the configuration file")
			return fmt.Errorf("database and server keys do not match, check the configuration file")
		}
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(dbList))

	for _, dbInfoList := range serversDatabases {
		dbListCopy := dbInfoList
		wg.Add(1)
		go func(dbInfos []DBInfo) {
			defer wg.Done()
			for _, dbInfo := range dbInfos {
				select {
				case <-a.ctx.Done():
					logging.L(a.ctx).Info("Backup cancelled by context")
					errCh <- fmt.Errorf("backup cancelled for database %s", dbInfo.Database.Name)
					return
				default:
					err := a.runBackup(dbInfo.Server, dbInfo.Database)
					if err != nil {
						logging.L(a.ctx).Warn(
							"Skip creating database",
							logging.StringAttr("name", dbInfo.Database.Name),
							logging.ErrAttr(err),
						)
						errCh <- err
						return
					}
				}
			}
		}(dbListCopy)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	logging.L(a.ctx).Info("All backups completed successfully")

	return nil
}

func (a *App) runBackup(server config.Server, db config.Database) error {
	dataFormat := utils.TemplateData{
		Server:   server.GetDisplayName(),
		Database: db.GetDisplayName(),
		Template: a.cfg.Settings.Template,
	}
	nameFile := utils.GetTemplateFileName(dataFormat)
	logging.L(a.ctx).Info("Generated template", logging.StringAttr("name", nameFile))

	cmdData := &cmdCfg.ConfigData{
		User:       db.User,
		Password:   db.Password,
		Name:       db.GetDisplayName(),
		Port:       db.GetPort(a.cfg.Settings.DBPort),
		Key:        server.SSHKey,
		Host:       server.Host,
		DumpName:   nameFile,
		DumpFormat: a.cfg.Settings.DumpFormat,
	}

	logging.L(a.ctx).Info("Prepare command for dump")

	cmdApp := command.NewApp(&a.cfg.Settings, cmdData)
	cmdStr, remotePath, err := cmdApp.GetCommand()
	if err != nil {
		logging.L(a.ctx).Error("failed to generate command")
		return fmt.Errorf("failed to generate command: %w", err)
	}

	logging.L(a.ctx).Info("Prepare connection")
	conn := connect.New(
		server.Host,
		server.User,
		server.GetPort(a.cfg.Settings.SrvPost),
		a.cfg.Settings.SSH.PrivateKey,
		server.SSHKey,
		a.cfg.Settings.SSH.Passphrase,
		server.Password,
		*a.cfg.Settings.SSH.IsPassphrase,
	)

	fmt.Println("Connecting to server...")
	if err := runWithCtx(a.ctx, conn.Connect); err != nil {
		logging.L(a.ctx).Error("Failed to connect to server")
		return err
	}

	defer func(conn *connect.Connect) {
		_ = conn.Close()
	}(conn)

	logging.L(a.ctx).Info("Trying to establish connection to server %s...\n", server.Host)
	if err := runWithCtx(a.ctx, conn.TestConnection); err != nil {
		logging.L(a.ctx).Error(
			"Failed to test connection to server",
			logging.StringAttr("server", server.Host),
			logging.ErrAttr(err),
		)
		return err
	}
	logging.L(a.ctx).Info("The connection has established")

	logging.L(a.ctx).Info("Preparing for backup creation")
	backupApp := backup.NewApp(a.ctx, conn, cmdStr, remotePath, a.cfg.Settings.DirDump, a.cfg.Settings.DumpLocation)

	if err := runWithCtx(a.ctx, backupApp.Backup); err != nil {
		logging.L(a.ctx).Error("Failed to create backup")
		return err
	}
	logging.L(a.ctx).Info("The backup was successfully created and downloaded")

	if a.cfg.Settings.DirArchived != "" {
		logging.L(a.ctx).Info("Search for old backups")
		dbNamePrefix := fmt.Sprintf("%s_%s", server.GetDisplayName(), db.GetDisplayName())

		if err := runWithCtx(a.ctx, func() error {
			return utils.ArchivedLocalFile(dbNamePrefix, remotePath, a.cfg.Settings.DirDump, a.cfg.Settings.DirArchived)
		}); err != nil {
			logging.L(a.ctx).Error("Failed to archive backups")
			return err
		}

		logging.L(a.ctx).Info("Archived old backups", logging.StringAttr("path", a.cfg.Settings.DirArchived))
	}

	return nil
}

func runWithCtx(ctx context.Context, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled")
	case err := <-done:
		return err
	}
}
