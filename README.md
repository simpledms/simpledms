# SimpleDMS
SimpleDMS is an easy-to-use document management system (DMS) for small businesses that sorts documents almost by itself.

The app is also available as a SaaS offering at [simpledms.eu](https://simpledms.eu) and [simpledms.ch](https://simpledms.ch).

![Screenshot of the SimpleDMS app](https://simpledms.ch/assets/simpledms/2025.01.22-simpledms-screenshot_metadaten.png)

## Open Source philosophy and business model
The SimpleDMS app in this repository contains all features relevant for the use by a single company or family. The goal is to keep all these features, including future features, available for free.

To prevent making competing the SimpleDMS SaaS offering (simpledms.eu / simpledms.ch) to easy, a control plane to manage multi-tenant setups (customer management, billing integration, per customer storage limits, maybe white-labeling in the future, etc.) is locked behind a paywall. The code in the paywalled repo is still AGPL-licensed, but can only be accessed for a monthly fee. 

In addition to the SaaS offering, there is a business offering to obtain the code under a non-copyleft license for 1 EUR / user / month. This offering also includes the right to remove the attribution notices.

On demand, paid access to a SimpleDMS version with long-term support (LTS) and support plans can be offered.

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

