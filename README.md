# Yet Another Standardfile Implementation in Go

[![CircleCI](https://circleci.com/gh/mdouchement/standardfile/tree/master.svg?style=shield)](https://circleci.com/gh/mdouchement/standardfile/tree/master)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/mdouchement/standardfile)
[![Go Report Card](https://goreportcard.com/badge/github.com/mdouchement/standardfile)](https://goreportcard.com/report/github.com/mdouchement/standardfile)
[![License](https://img.shields.io/github/license/mdouchement/standardfile.svg)](http://opensource.org/licenses/MIT)

This is a Golang implementation of the [Standard File](https://standardfile.org/) protocol.

### Running your own server

You can run your own Standard File server, and use it with any SF compatible client (like Standard Notes).
This allows you to have 100% control of your data.
This server implementation is built with Go and can be deployed in seconds.

https://hub.docker.com/r/mdouchement/standardfile

### Client library

Go to `pgk/libsf` for more details.
https://godoc.org/github.com/mdouchement/standardfile/pkg/libsf

It is an alternative to https://github.com/jonhadfield/gosn

### SF client

```sh
go run cmd/sfc/main.go -h
```

Terminal UI client:
![sfc note](https://user-images.githubusercontent.com/6150317/62490536-c997f780-b7c9-11e9-867a-bc619d286b31.png)

## Requirements

- Golang 1.11.x (Go Modules)

### Technologies / Frameworks

- [Cobra](https://github.com/spf13/cobra)
- [Echo](https://github.com/labstack/echo)
- [BoltDB](https://github.com/etcd-io/bbolt) + [Storm](https://github.com/asdine/storm) Toolkit
- [Gowid](https://github.com/gcla/gowid)


## Differences with reference implementation

<details>
<summary>Drop legacy support for clients which hardcoded the "api" path to the base url (iOS)</summary>

> [Permalink](https://github.com/standardfile/ruby-server/blob/0a48c2625afc21966b110e0f73a1ff7bd212dbf4/config/routes.rb#L19-L26)

</details>

<details>
<summary>Drop the POST request done on Extensions (backups too)</summary>

> [Permalink](https://github.com/standardfile/ruby-server/blob/09b2020313a54668b7c6c0e122bbc8a530767d06/app/controllers/api/items_controller.rb#L20-L45)

This feature is pretty undocumented and I feel uncomfortable about the outgoing traffic from my server on unknown URLs.

</details>

<details>
<summary>Drop V1 support</summary>

> [All stuff used in v1 and not in v2 nor v3](https://github.com/standardfile/standardfile.github.io/blob/master/doc/spec-001.md)

</details>

<details>
<summary>JWT revocation strategy after password update</summary>

> Reference implementation use a [pw_hash](https://github.com/standardfile/ruby-server/blob/0a48c2625afc21966b110e0f73a1ff7bd212dbf4/app/controllers/api/api_controller.rb#L37-L43) claim to check if the user has changed their pw and thus forbid them from access if they have an old jwt.

<hr>

> Here we will revoke JWT based on its `iat` claim and `User.PasswordUpdatedAt` field.
> Looks more safer than publicly expose any sort of password stuff.
> See `internal/server/middlewares/current_user.go`

</details>

## Not implemented (yet)

- **2FA** (aka `verify_mfa`)
- Postgres if a more stronger database is needed
- A console for admin usage


## License

**MIT**


## Contributing

All PRs are welcome.

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request
