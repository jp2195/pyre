# Changelog

## [1.5.0](https://github.com/jp2195/pyre/compare/v1.4.0...v1.5.0) (2026-06-13)


### Features

* security hardening, view-engine consolidation, and test coverage ([#32](https://github.com/jp2195/pyre/issues/32)) ([b6b204e](https://github.com/jp2195/pyre/commit/b6b204e74c8cc1236b470f349b98754fd62e8d72))

## [1.4.0](https://github.com/jp2195/pyre/compare/v1.3.0...v1.4.0) (2026-05-22)


### Features

* initial pyre TUI with CI/CD and release automation ([1bc4dc4](https://github.com/jp2195/pyre/commit/1bc4dc4248b4c7dfd04947485de775f5367819bf))
* **tui:** add 10 navigation and view enhancements ([d9b66df](https://github.com/jp2195/pyre/commit/d9b66dff5300d9c75cce6b5346106e08c44159e5))
* **tui:** add IPSec Tunnels & GP Users views, UX improvements ([9a0198b](https://github.com/jp2195/pyre/commit/9a0198b952107f58bdca2e7777780c0af811c464))
* **tui:** add IPSec Tunnels & GP Users views, UX improvements ([931cce5](https://github.com/jp2195/pyre/commit/931cce5c85c3b106ae0abb1cada59e0b93d31598))


### Bug Fixes

* Add the d selector to the help menu when utilizing panorama. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))
* **ci:** add packages config for release-please ([ff256b0](https://github.com/jp2195/pyre/commit/ff256b0cf872683bdfc19bc55d969a402f87954d))
* **ci:** add release please token ([aac0166](https://github.com/jp2195/pyre/commit/aac0166077e39fea29e8406299d788dd57fe2865))
* **ci:** pin Go version to 1.25.7 across all workflows ([f2830a4](https://github.com/jp2195/pyre/commit/f2830a47cc1e52c05b5f02d56cdd48cd9ff7e865))
* **ci:** use merge-multiple for artifact download ([402a8fb](https://github.com/jp2195/pyre/commit/402a8fb78d87c9224e593f398350d75b9bf8dd53))
* **deps:** update module golang.org/x/crypto to v0.48.0 ([#17](https://github.com/jp2195/pyre/issues/17)) ([8c0da4a](https://github.com/jp2195/pyre/commit/8c0da4a9f697f5aecbd4681a689c1204dd089d6b))
* **lint:** resolve golangci-lint errors ([59b6513](https://github.com/jp2195/pyre/commit/59b6513794c9b9808bd6e45acb7d7e1874bf0bed))
* **lint:** resolve golangci-lint errors and remove troubleshooting ([d9a9452](https://github.com/jp2195/pyre/commit/d9a9452e9a1ab5df40425a15aca198359cc8c6f4))
* Panorama default/entry screen to the device selector. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))
* resolve lint errors and pin CI Go version to 1.25.7 ([66809e2](https://github.com/jp2195/pyre/commit/66809e2670f65a40e7598cdea32149632461c7e3))
* space bar to select/toggle and enter to move to the next form. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))
* trigger release for refactor and docs work ([600b33e](https://github.com/jp2195/pyre/commit/600b33ecec1503d36068b04b60a9f114ca69262c))

## [1.3.0](https://github.com/jp2195/pyre/compare/v1.2.1...v1.3.0) (2026-05-22)


### Features

* **api:** add Address and Service objects to PAN-OS client, fetching from `/vsys1` and `/shared` with merged scopes; supports all four AddressObject variants (ip-netmask, ip-range, fqdn, ip-wildcard) and preserves Service port strings ([#25](https://github.com/jp2195/pyre/pull/25))
* **views:** add ObjectsModel with per-tab Address/Service state, per-tab cursor/filter/sort, and Tab/`a`/`s` switching ([#25](https://github.com/jp2195/pyre/pull/25))
* **tui:** wire Objects view into navigation, dispatch, and refresh (AddressesMsg/ServicesMsg, fetchObjects batch, Analyze nav group between NAT and Sessions) ([#25](https://github.com/jp2195/pyre/pull/25))
* **tui:** add Objects to command palette and document its keys in `docs/keybindings.md` ([#25](https://github.com/jp2195/pyre/pull/25))


### Bug Fixes

* **views/interfaces:** stop slicing into mid-rune of the 3-byte status indicator ([#25](https://github.com/jp2195/pyre/pull/25))
* **api/policies:** `parseRuleHitCounts` wraps inner XML to match every other call site ([#25](https://github.com/jp2195/pyre/pull/25))
* **api/logs:** `pollLogJob` polls before sleeping; saves ~500ms per query ([#25](https://github.com/jp2195/pyre/pull/25))
* **auth:** handle `:port` and IPv6 forms in `PYRE_<HOST>_API_KEY` normalization ([#25](https://github.com/jp2195/pyre/pull/25))
* **views/dashboard_vpn:** route up-bar color through the theme system ([#25](https://github.com/jp2195/pyre/pull/25))
* **views/command_palette:** replace handrolled itoa (overflowed `MinInt`) with `strconv.Itoa` ([#25](https://github.com/jp2195/pyre/pull/25))
* **api/monitoring:** sanitize disk-usage fields before TUI display ([#25](https://github.com/jp2195/pyre/pull/25))
* **views/connection_hub:** align cursor with sorted slice (was off for interleaved Panorama entries) ([#25](https://github.com/jp2195/pyre/pull/25))
* **config:** atomic write for `~/.pyre.yaml.bak` ([#25](https://github.com/jp2195/pyre/pull/25))


### Code Refactoring

* concurrency: lock `auth.Connection` mutable fields (`IsPanorama`, `ManagedDevices`, `TargetSerial`) behind `sync.RWMutex` with accessor methods; race regression test added ([#25](https://github.com/jp2195/pyre/pull/25))
* TUI cleanup: collapse duplicate helpers (single canonical `formatNumberWithCommas`, `formatTimeAgo` with weeks tier, `SeverityStyle`); adopt `TableBase.VisibleRows`/`EnsureCursorValid`/`EnsureVisible` across views; `setError` returns `(Model, tea.Cmd)`; drop unused `DashboardSelectedMsg` ([#25](https://github.com/jp2195/pyre/pull/25))
* modernize: `go fix` sweep (range-N loops, `min`/`max`, `maps.Copy`); migrate `errors.As` to `errors.AsType[T]` ([#25](https://github.com/jp2195/pyre/pull/25))
* visible UX: Connection Hub renders a single CONNECTIONS section with inline `[Firewall]`/`[Panorama]` tags in recency order; hit-count "Last Hit" gains `Xw ago` tier; informational log severity now uses `SeverityInfoStyle` instead of muted gray ([#25](https://github.com/jp2195/pyre/pull/25))
* remove orphaned `internal/troubleshoot` package and `views/troubleshoot.go` (never wired into `app.go`; required SSH access that has been removed) ([#25](https://github.com/jp2195/pyre/pull/25))
* **config:** remove unused `Settings.DefaultView` field — was parsed but no caller ever read it; start view is chosen by `cmd/pyre.determineStartView` ([#26](https://github.com/jp2195/pyre/pull/26))


### Documentation

* consolidate docs tree, fix stale references, add missing per-view docs for Objects/Routes/IPSec/GP Users plus a `docs/views/README.md` index; align Analyze nav listings across README/getting-started/keybindings ([#26](https://github.com/jp2195/pyre/pull/26))
* add a "Your first 60 seconds" walkthrough to getting-started; collapse duplicated navigation reference so `docs/keybindings.md` is the single source of truth ([#26](https://github.com/jp2195/pyre/pull/26))
* restructure README with centered hero, screenshot slots, condensed install, and a tip callout for credential handling ([#26](https://github.com/jp2195/pyre/pull/26))
* drop duplicated install / Connection Hub key sections from `getting-started.md` and `configuration.md` in favor of pointers to the authoritative pages ([#26](https://github.com/jp2195/pyre/pull/26))


### Miscellaneous Chores

* **deps:** bump Go to 1.26.3 for stdlib CVE patches (GO-2026-4971 `net.Dial`/`LookupPort` NUL-byte panic; GO-2026-4918 HTTP/2 transport infinite loop) ([#25](https://github.com/jp2195/pyre/pull/25))
* **lint:** bring repo to a lint-clean baseline for golangci-lint v2.11.4 (gofmt, staticcheck QF1008/QF1012/QF1034, prealloc, scoped gosec exclusions on test fixtures) ([#25](https://github.com/jp2195/pyre/pull/25))
* **ci:** pin security-tool installs (`gosec` via `go install @v2.22.11`, `govulncheck @v1.3.0`) and add a baseline 3-day Renovate `minimumReleaseAge` with `vulnerabilityAlerts` opt-out ([#27](https://github.com/jp2195/pyre/pull/27))
* **deps:** update `actions/dependency-review-action` to v5 ([#23](https://github.com/jp2195/pyre/pull/23))
* **deps:** update github actions ([#24](https://github.com/jp2195/pyre/pull/24))

## [1.2.1](https://github.com/jp2195/pyre/compare/v1.2.0...v1.2.1) (2026-04-19)


### Bug Fixes

* **deps:** update module golang.org/x/crypto to v0.48.0 ([#17](https://github.com/jp2195/pyre/issues/17)) ([8c0da4a](https://github.com/jp2195/pyre/commit/8c0da4a9f697f5aecbd4681a689c1204dd089d6b))

## [1.2.0](https://github.com/jp2195/pyre/compare/v1.1.1...v1.2.0) (2026-02-09)


### Features

* **tui:** add IPSec Tunnels & GP Users views, UX improvements ([9a0198b](https://github.com/jp2195/pyre/commit/9a0198b952107f58bdca2e7777780c0af811c464))
* **tui:** add IPSec Tunnels & GP Users views, UX improvements ([931cce5](https://github.com/jp2195/pyre/commit/931cce5c85c3b106ae0abb1cada59e0b93d31598))


### Bug Fixes

* **ci:** pin Go version to 1.25.7 across all workflows ([f2830a4](https://github.com/jp2195/pyre/commit/f2830a47cc1e52c05b5f02d56cdd48cd9ff7e865))
* resolve lint errors and pin CI Go version to 1.25.7 ([66809e2](https://github.com/jp2195/pyre/commit/66809e2670f65a40e7598cdea32149632461c7e3))

## [1.1.1](https://github.com/jp2195/pyre/compare/v1.1.0...v1.1.1) (2026-02-03)


### Bug Fixes

* Add the d selector to the help menu when utilizing panorama. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))
* Panorama default/entry screen to the device selector. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))
* space bar to select/toggle and enter to move to the next form. ([b3ad73c](https://github.com/jp2195/pyre/commit/b3ad73c3674ca5542c04b48f8b9c9689d98a512c))

## [1.1.0](https://github.com/jp2195/pyre/compare/v1.0.1...v1.1.0) (2026-01-27)


### Features

* **tui:** add 10 navigation and view enhancements ([d9b66df](https://github.com/jp2195/pyre/commit/d9b66dff5300d9c75cce6b5346106e08c44159e5))

## [1.0.1](https://github.com/jp2195/pyre/compare/v1.0.0...v1.0.1) (2026-01-25)


### Bug Fixes

* **ci:** use merge-multiple for artifact download ([402a8fb](https://github.com/jp2195/pyre/commit/402a8fb78d87c9224e593f398350d75b9bf8dd53))

## 1.0.0 (2026-01-25)


### Features

* initial pyre TUI with CI/CD and release automation ([1bc4dc4](https://github.com/jp2195/pyre/commit/1bc4dc4248b4c7dfd04947485de775f5367819bf))


### Bug Fixes

* **ci:** add packages config for release-please ([ff256b0](https://github.com/jp2195/pyre/commit/ff256b0cf872683bdfc19bc55d969a402f87954d))
* **ci:** add release please token ([aac0166](https://github.com/jp2195/pyre/commit/aac0166077e39fea29e8406299d788dd57fe2865))
* **lint:** resolve golangci-lint errors ([59b6513](https://github.com/jp2195/pyre/commit/59b6513794c9b9808bd6e45acb7d7e1874bf0bed))
* **lint:** resolve golangci-lint errors and remove troubleshooting ([d9a9452](https://github.com/jp2195/pyre/commit/d9a9452e9a1ab5df40425a15aca198359cc8c6f4))
