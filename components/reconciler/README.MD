# Reconciler

## Overview

This module allows communication with the Reconciler's main component by generating the Mothership REST Client.

The Mothership REST Client is generated
from [this](https://raw.githubusercontent.com/kyma-incubator/reconciler/569388cbdde10092e5cdab25d5d87cf6a038a4d2/openapi/external_api.yaml)
OpenAPI spec.

## Prerequisites

- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
- [mockgen](https://github.com/golang/mock)

## Generate the Mothership REST Client

To generate the Mothership REST Client run:

```bash
    make all
```
