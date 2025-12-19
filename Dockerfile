# must match debian version below for libicu-dev compatibility
# during build and runtime
FROM golang:1.25-bookworm AS build-env

# build must be run with `docker build --ssh default .`
# to make use of `--mount=type=ssh` below

WORKDIR /usr/src/simpledms

# ENV GOPRIVATE=github.com/marcobeierer

RUN mkdir ~/.ssh
RUN chmod 700 ~/.ssh
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts

#RUN git config --global user.name "marco"
#RUN git config --global user.email "email@marcobeierer.com"
#RUN git config --global url."ssh://git@github.com/marcobeierer/".insteadOf "https://github.com/marcobeierer/"

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN --mount=type=ssh go mod download && go mod verify

RUN apt update
# libicu-dev is required for sqlite_icu during build and when running
RUN apt install -y libicu-dev

COPY . .
RUN CGO_ENABLED=1 go build -v -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" -o /usr/local/bin/simpledms .
#RUN go build -v -o /usr/local/bin/app ./...

# having a separate build-env, for example to not include source files or testing databases
# in final image;
# also using bookworm-slim for the final build leads to very small (100 MB) images
#
# must match golang debian version above for libicu-dev compatibility
# during build and runtime
FROM debian:bookworm-slim

RUN apt update
RUN apt upgrade
# libicu-dev is required for sqlite_icu during build and when running
RUN apt install -y ca-certificates libicu-dev
RUN update-ca-certificates

# Copy over the built binary from our builder stage
COPY --from=build-env /usr/local/bin/simpledms /usr/local/bin/simpledms

# used for hosting files
VOLUME /srv

# necessary because serving a specific dir is not supported yet
WORKDIR /srv

# meta path is relative to WORKDIR
CMD ["/usr/local/bin/simpledms"]
