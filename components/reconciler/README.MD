# Reconciler

## Overview

This module allows making communication with the Reconciler's main component by generating The Mothership REST Client.

The Mothership REST Client is generated from [this](https://raw.githubusercontent.com/kyma-incubator/reconciler/569388cbdde10092e5cdab25d5d87cf6a038a4d2/openapi/external_api.yaml) OpenAPI spec.

## Prerequisites

- [oapi-codegen](https://github.com/deepmap/oapi-codegen)

## Generate Client

To generate the Mothership REST Client run:

```bash
    make generate
```
