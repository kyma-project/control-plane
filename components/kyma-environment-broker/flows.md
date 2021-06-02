### Scenario 1
NOTE: consider field name like oidcOverride


```json
{
  "name": "my-cluster",
  "region": "eu-central-1"
}
```
The default OIDC params are used.

Update:

```json
{
  "name": "my-cluster",
  "region": "eu-central-1",
  "oidc": {
     "clientID":"client-id-001",
     "issuerURL":"https://issuer.url",  
     "signingAlgs":["RSA256"]
  }
}
```
New OIDC goes to Provisioner.

Second update:

```json
{
  "name": "my-cluster",
  "region": "eu-central-1",
   "oidc": {
   "default": true
   }
}
```
OIDC override is empty, then the defaault one is taken into account.

Pros: simple contract
Cons: KEB is aware of default OIDC values
 
### Scenario 2
KEB is not aware of defaults.

```json
{
  "name": "my-cluster",
  "region": "eu-central-1"
}
```

The default OIDC params are used.

Update:

```json
{
  "name": "my-cluster",
  "region": "eu-central-1",
  "oidc": {
     "clientID":"client-id-001",
     "issuerURL":"https://issuer.url",  
     "signingAlgs":["RSA256"]
  }
}
```

second update:

```json
{
  "name": "my-cluster",
  "region": "eu-central-1"
}
```
KEB does not send oidc, provisioner use defaults

Pros: simple KEB

Cons Provisioner works different way with oidc than other fields
 
 