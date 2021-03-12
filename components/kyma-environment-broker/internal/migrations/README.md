## Migrations

Whenever we want to execute migration which requires breaking changes, and a proper migration this package can be used to store migration files.

### Best practices

- For each migration which requires breaking changes there should be implemented a rollback which can be used to revert changes in case of any problems
- Migrations and rollbacks execution must be configurable
- Before the migration both scenarios should be tested on the mocked postgres storage containing the same data as on the DEV environment
- After the migration is done on every environment we should get rid of them from the KEB's source code
