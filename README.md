# Kānuka CLI Tool

Kānuka is a CLI tool written in Go. It makes sharing secrets, creating
development environments, deploying code, and testing, unified under one robust
and simple interface. Never again will you need to ask another developer to
send you their `.env` file, or what tools need to be installed on your system
in order to start developing.

## What's with the name?

Kānuka (Kunzea eicoides) is a tree that is native and endemic to New Zealand.
It is a robust species that is critical to restoring wildlife that was
destroyed in a fire, as it quickly propagates and regenerates the land. Its
leaves has a characteristically soft touch, and is one of the few plants that
can survive the heat of geothermal features.

It is fast, resilient, yet pleasant to touch. This is the vision of Kānuka.

## Feature roadmap

All features should be driven with the idea that things should be _fast_,
_robust/secure_, and _easy to use_. There are many common pain points with
developers. `kanuka` wants to tackle the following:

1. It isn't easy to share a `.env` file. A developer could make a change on one
   machine, and by its very nature, that change will **not** reflect on another
   developer's machine. Without third party services, it also becomes taxing for a
   developer to share secrets. Let's be honest. Who hasn't sent an API key over a
   messaging service?

2. Developers can use a package manager, but what if a project now depends on
   something that isn't explicitly a part of the source code? For example, you
   might have a Python project, but that depends on a developer to have Docker
   installed in order to run tests properly. Docker can't be defined in a Python
   lockfile. But what if it didn't need to be? What if you ran `kanuka shell`,
   knowing with full confidence that no matter what machine your developer has,
   you have the exact same environment down to the operating system?

3. One project, you might need to run `pytest`, and another, you might need to
   run `npx jest` or `jest` (depending on the developer's machine), and on yet
   another, `gradle test` or `mvn test`. Some tests need special configuration,
   such as setting a Python path before running the test. Sometimes you need to
   call your package manager before you run the test. It is different every time.
   But what if this was unified under one simple `kanuka test`?

4. While infrastructure-as-code paradigms have certainly simplified deployment,
   and made the entire process reproducible, running the environment still
   often requires installing packages that are separate to the source code of your
   project. What if we just had `kanuka deploy@development`, `kanuka
deploy@staging`, and `kanuka deploy@production`, with all the secrets and
   configuration handled for you?

These points are ordered in priority. Therefore, at this stage of the project,
the first goal is to achieve feature number 1.

### Secrets management

The idea of secrets management is this: Encrypt an `.env` file using a
symmetric key, and then have the symmetric encrypted with a public/private key
pair. The new `.env.kanuka` and the `{users_key}.kanuka` should be committed with the
project. For example:

```bash
root
├─ env.kanuka
├─ .kanuka/secrets/
│  ├─ user_1.kanuka
│  ├─ user_2.kanuka
│  └─  user_3.kanuka
├─ src/
└─ ...
```

- [ ] Decide on if there should be an owner/worker structure with the key
      distribution, or if there should be a flat encryption structure, where anybody
      with decrypt access can add another user.

- [ ] Search through the entire project for all `*.env*` files, and encrypt
      them automatically with a `kanuka secrets encrypt` command, and `kanuka secrets
decrypt` for the reverse process.

- [ ] Functionality to create and add another user's encrypted symmetric key
      with `kanuka secrets add --file {path_to_pubkey_file} --username {username}`
      (or decide on having just randomised encrypted key names).

- [ ] Have a `kanuka secrets init`, which automatically generates a
      public/private key pair for you in a `~/.kanuka/keys/` directory, and then gives
      instructions for how to get your encrypted symmetric key into the repo. If you
      are the first user to run the `init` command, then automatically generate the
      encrypted symmetric key and encrypt all discovered `*.env*` files.

- [ ] Have a `kanuka secrets remove` command, which will remove your own key
      from the repo and your `~/.kanuka/keys/` directory.

- [ ] Have a `kanuka secrets purge` command, which will delete every single key
      in the git history (in the event that someone's private key got leaked).

- [ ] Have a `.kanuka_settings.yaml` which will define any specialties such as
      the regex for what files to encrypt, or the default encryption algorithm, for
      example.

### Shell management

Still TBD.

### Testing management

Still TBD.

### Deployment management

Still TBD.

## Project setup

To get started, you must have go>=1.23 installed. To build the project, run the
following command:

```bash
# installs all missing packages and removes all unused packages
go mod tidy
```

To test your changes, ensure that `$GOBIN` is in your `PATH` by adding this
line to your `.bashrc` or `.zshrc` (or whatever other interpreter you use):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

And then to build the package, make sure you are in the root directory of the
project, and run this command:

```bash
# builds the binary and places it in your $GOBIN
go install
# run the binary
kanuka

# alternatively, if you'd rather the binary be local
go build
./kanuka
```
