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

We accepted timestamps with yyyyMMddHHmm format at the beginning of the file name as a standard naming convention. However, due to mistake, some of the Provisioner's migration files are named with yyyyddMMHHmm timestamp format. To ensure that new files are in right order, please follow the workaround - until the end of 2021 add 31 to the month, so use the yyyy(MM+31)ddHHmm format.