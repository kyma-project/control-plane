# Schema Migrator

## Overview

The Schema Migrator is responsible for database schema migrations.

## Development

If you want to modify database schema used by compass, add migration files (follow [this](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) instructions) to `migrations` directory. 
New image of migrator will be produced that contains all migration files so make sure to bump component version value in compass chart.
To test if migration files are correct, execute:
```
make verify
```

## Naming convention

Originally, we accepted timestamps with the `yyyyMMddHHmm` format at the beginning of the file name as the standard naming convention. However, due to a mistake, some of the Runtime Provisioner's migration files were named using the `yyyyddMMHHmm` timestamp format. **To ensure that new files are in the right order, until the end of 2021, follow the workaround `yyyy(MM+31)ddHHmm` pattern, which adds 31 to the month number.**
