# Migrations

If you want to provide a migration that requires breaking changes, you can use this package to store the migration files.

## Best practices

- For each migration that requires breaking changes, you should implement a rollback that easily reverts changes in case of any problems.
- Migration and rollback execution must be configurable.
- Before the migration, both migration and rollback scenario should be tested on the mocked Postgres storage containing the same data as on the `DEV` environment.
- After the migration is done on every environment, remove it from the KEB's source code.
