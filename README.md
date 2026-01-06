# SimpleDMS – Document management for small businesses
SimpleDMS is an easy-to-use document management system (DMS) for small businesses that sorts documents almost by itself.

## Simple and efficient
The metadata-driven concept of SimpleDMS enables efficient filing and fast retrieval of documents after a short familiarization period.

The core of the concept consists of:

- a well thought-out tag system for categorizing documents, and
- workspaces (spaces) for shared or private access to documents.

## SaaS
The app is also available as a SaaS offering at [simpledms.eu](https://simpledms.eu) and [simpledms.ch](https://simpledms.ch).

## Screenshot
![Screenshot of the SimpleDMS app](https://simpledms.ch/assets/simpledms/2025.01.22-simpledms-screenshot_metadaten.png)

## Open Source philosophy and business model
The SimpleDMS app in this repository contains all features relevant for the use by a single company or family. The goal is to keep all these features, including future features, available for free.

To prevent making competing the SimpleDMS SaaS offering (simpledms.eu / simpledms.ch) to easy, a control plane to manage multi-tenant setups (customer management, billing integration, per customer storage limits, maybe white-labeling in the future, etc.) is locked behind a paywall. The code in the paywalled repo is source-available and modification is allowed.

In addition to the SaaS offering, there are business offerings to obtain the source code under a non-copyleft license or a license that allows removing the attribution notices in the app.

On demand, paid access to a SimpleDMS version with long-term support (LTS) and support plans can be offered.

You can find the [prices on the SimpleDMS website](https://simpledms.eu/en/pricing).

## Tech stack
SimpleDMS is built with:
- [Go](https://go.dev/)
- [ent](https://entgo.io/) Entity Framework
- [SQLite](https://sqlite.org/)
- [htmx](https://htmx.org/) with Go templates
- [Tailwind CSS](https://tailwindcss.com/)

## Where is the git history?
In the beginning SimpleDMS was developed in a monorepo together with other apps as a closed source project.

When open-sourcing SimpleDMS in December 2025, I decided to remove the git history, because preserving it while extracting the project from the monorepo was not worth the effort. In addition, I didn't want to risk exposing any personal notes or details of my other projects.

