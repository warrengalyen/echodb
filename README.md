# EchoDB

A CLI utility for creathing echos (backups) of various databases types with configurable connection and storage settings.

## Features

- Supports **PostgreSQL**, **MySQL** databases.
- Direct connection (dump performed directly on the server and downloaded)=
- SSH support
- Custom dump name templates.
- Configuration file support.
- Archiving backups
- Backup formats:
    - PostgreSQL: `plain`, `dump`, `tar`

## Configuration

The configuration is set in a YAML file. (e.g. `config.yaml`)

---

### ⚙️ Configuration Example

```yaml
settings"
  db_port: "5432"
  driver: "psql"
  ssh:
    private_key: "your_key_path"
    passphrase: "your_passphrase"
    is_passphrase: true
  template: "{%srv%}_{%db%}_{%time%}"
  archive: true
  location: "server"
  format: "plain"
  dir_dump: "./dumps"
  dir_archived: "./archived"

servers:
  test:
    name: "test server"
    host: "127.0.0.1"
    port: "22"
    user: "user"
    password: "password"

databases:
  test_demo:
    name: "demo"
    user: "user
    password: "password
    server: "test
    port: "5432"
    driver: "psql"

  test_app:
    user: "app"
    password: "pass"
    server: "test"
```

---

### 📑 Configuration Description

#### The configuration consists of three sections

#### 🔧 1. Settings — Global Settings

Apply to all servers and databases, unless redefined locally.

| Parameter           | Description                                                                               | is        |
|---------------------|-------------------------------------------------------------------------------------------|-----------|
| `db_port`           | Default database connection port                                                          | option    |
| `driver`            | The default DB driver: `psql`                                                             | required  |
| `ssh.private_key`   | The path to the private SSH key.                                                          | option    |
| `ssh.passphrase`    | Passphrase for the key (optional).                                                        | option    |
| `ssh.is_passphrase` | whether to use passphrase from the config                                                 | option    |
| `template`          | File Name Template: `{%srv%}`, `{%db%}`, `{%datetime%}`, `{%date%}`, `{%time%}`, `{%ts%}` | option    |
| `archive`           | Archiving old dumps (need `{%srv%}_{%db%}` in template).                                  | option    |
| `location`          | Dump execution method: `server`                                                           | required  |
| `format`            | Dump format: `plain`, `dump`, `tar`.                                                      | required  |
| `dir_dump`          | Directory for saving dumps                                                                | option    |
| `dir_archived`      | Archive Directory                                                                         | option    |

#### Params

- #### template

  - `{%srv%}` —  Name server
  - `{%db%}` —  Name db
  - `{%datetime%}` —  Date and time
  - `{%date%}` — Date
  - `{%time%}` — Time
  - `{%ts%}` — Time unix

- #### location

  - `server` — create dump in server and download

- #### format

  - PostgreSQL: `plain`, `dump`, `tar`

#### 🖥 2. Servers

Defines the connections through which databases can be backed up.

| Parameter   | Description                         | is                                     |
|-------------|-------------------------------------|----------------------------------------|
| `name`      | Human-readable server name          | option                                 |
| `host`      | The IP address or domain name       | required                               |
| `port`      | Connection port                     | required<br/> (if not set global)      |
| `user`      | Username.                           | required                               |
| `password`  | Password (if there is no key)       | required<br/> (if not set key)         |

#### 🗄 3. Databases

A list of databases that need to be backed up.

| Parameter   | Description                                            | is                                |
|-------------|--------------------------------------------------------|-----------------------------------|
| `name`      | Database name (by default, the key name)               | option                            |
| `user`      | The database user                                      | required                          |
| `password`  | DB user's password                                     | required                          |
| `server`    | The link to the server from the `servers` section      | required                          |
| `port`      | Connection port (if different from `settings.db_port`) | required<br/> (if not set global) |
| `driver`    | driver: `psql`                                         | required<br/> (if not set global) |
---

### ▶ Launch examples

#### Backup with a choice of database from config file

```bash
./echodb
````

#### Backup with a choice of database with set config file

```bash
./echodb --config ./config.yaml
````

### 📂 Application structure

```bash
├── dumps/       # Directory for new dumps
├── archived/    # Archive of old dumps
├── config.yaml  # Configuration file
└── echodb       # The executable file of the utility
```
