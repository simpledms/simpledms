# Contributing

SimpleDMS is selling exceptions to the AGPL. To make this possible, it is necessary for contributors to sign a Contribution License Agreement (CLA).

You can find more information on the business model of SimpleDMS in the README:
https://github.com/simpledms/simpledms?tab=readme-ov-file#open-source-philosophy-and-business-model

An article from Richard Stallman on the practice of selling exceptions to the GPL can be found on the GNU:
https://www.gnu.org/philosophy/selling-exceptions.html.en

The CLA is based on the Individual Contributor License Agreement v2.2 by the Apache Software Foundation. You can find the CLA in the repository:
https://github.com/simpledms/simpledms/blob/main/.cla/v1/cla.md

Please let me know if you have any questions about the CLA or have any concerns about it. I'm open for discussions as I want to find the best solution for SimpleDMS and potential contributors.

## Setup Dev Environment

### Required Tools
- Go
	- see go.mod for required version
	- run the following if your Linux distro doesn't come with the required version:
		- `go install golang.org/dl/go1.25.5@latest; go1.25.5 download`
		- you then have to setup an alias or use `go1.25.5` instead of `go` command.
- direnv
- TailwindCSS
- Node.js
- air
- Docker
	- Docker Compose (comes with newer Docker versions, separate installation for older versions)

```
git clone git@github.com:simpledms/simpledms.git
cd simpledms
cp .env.sample .env # edit .env
# edit .env file
direnv allow
docker compose up -d # docker-compose up -d for older Docker versions
# spins up minio, tika and mailpit
go tool air
```
