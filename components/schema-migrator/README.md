# Schema Migrator

## Overview

The Schema Migrator is responsible for Runtime Provisioner's database schema migrations.

## Development

If you want to modify database schema used by Compass, add migration files to `/resources/kcp/charts/<component>/migrations` directory. To do it, follow the instructions in the [Migrations](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) document. The new migration files are mounted as ConfigMaps. Make sure to bump the version of the component for which the migration was added.
To test if the migration files are correct, run:
```
make verify
```

> **CAUTION** : The following method of adding migrations files is deprecated:\
*If you want to modify database schema used by Compass, add [migration files](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) to `migrations` directory. 
New image of migrator will be produced that contains all migration files so make sure to bump component version value in Compass chart.*

## Naming convention

Originally, we accepted timestamps with the `yyyyMMddHHmm` format at the beginning of the file name as the standard naming convention. However, due to a mistake, some of the Runtime Provisioner's migration files were named using the `yyyyddMMHHmm` timestamp format. **To ensure that new files are in the right order, until the end of 2021, follow the workaround `yyyy(MM+31)ddHHmm` pattern, which adds 31 to the month number.**
