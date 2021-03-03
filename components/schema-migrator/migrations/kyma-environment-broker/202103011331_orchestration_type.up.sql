ALTER TABLE orchestrations
  ADD COLUMN [type] varchar(32) NOT NULL DEFAULT "upgradeKyma";
