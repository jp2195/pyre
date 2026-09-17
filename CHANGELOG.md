# Changelog

## [1.6.0](https://github.com/jp2195/pyre/compare/v1.5.4...v1.6.0) (2026-09-17)


### Features

* **api:** add LogQuery and LogPage for log fetches ([3d348fd](https://github.com/jp2195/pyre/commit/3d348fdbbf56419b8f4907aefd52a999888f2823))
* **logs:** add a PAN-OS query bar on f ([7c1bb78](https://github.com/jp2195/pyre/commit/7c1bb7874e9d1baffe7ac205ecee0a83c766b0ef))
* **logs:** add time-range presets on t and T ([39c6887](https://github.com/jp2195/pyre/commit/39c6887fe9ad662f630577d5e12fab194f5f873a))
* **logs:** fetch only the tab that is on screen ([7632310](https://github.com/jp2195/pyre/commit/7632310e83f1547c08d77c3cc643919a3f2693bf))
* **logs:** name the query when nothing matched ([551d70e](https://github.com/jp2195/pyre/commit/551d70e3a7d1a32ac191bd4d77f4251c983480bc))
* **logs:** page through results with m ([a7d5d1f](https://github.com/jp2195/pyre/commit/a7d5d1f3296e922f732e640b3c5ec8776edf2a00))


### Bug Fixes

* **api:** interpret device timestamps in the device's zone ([ed3ac8c](https://github.com/jp2195/pyre/commit/ed3ac8c9075f3f20789cd51049ea84edf32dc725))
* **api:** keep PAN-OS parameter-error text ([2b0f888](https://github.com/jp2195/pyre/commit/2b0f88897521a793654bbbfcebf730115cc2d5d5))
* **api:** learn the device clock before writing a log time bound ([05a8702](https://github.com/jp2195/pyre/commit/05a8702789c06ac402731d0143b488f9e6870d5c))
* **api:** read certificates from the config without the private key ([1143607](https://github.com/jp2195/pyre/commit/114360757217859885a03143c3224f16cda7edba))
* **api:** read plain-text responses through their CDATA wrapper ([821c80e](https://github.com/jp2195/pyre/commit/821c80eeb43f12f2158938cc1bba332bc50a39eb))
* **api:** repair threat logs and rebuild the Threats panel from them ([261832e](https://github.com/jp2195/pyre/commit/261832edd9fc1f193d1c48fe363ff3d392df91dc))
* **api:** sanitize every decoded string, and strip C1 and bidi characters ([ebd47b1](https://github.com/jp2195/pyre/commit/ebd47b1be488386fd3a8343cb34b17ae92e2ce3d))
* **api:** trace the PAN-OS message the user already sees ([b48bb9d](https://github.com/jp2195/pyre/commit/b48bb9dff967955f11a84f9516d1a560d7c20d0a))
* **auth:** reject a URL pasted into the host field ([4587274](https://github.com/jp2195/pyre/commit/458727443de45cd75129a53c1d7fed72e034fd6e))
* **auth:** resolve -c credentials against the selected connection ([fa47c63](https://github.com/jp2195/pyre/commit/fa47c63bc966f7986c3941b03e7ae84c23323062))
* **config:** make permission warnings visible and honor --config on save ([e7c12ed](https://github.com/jp2195/pyre/commit/e7c12ed71eaa19693be8e323ae6db94a7c4f37cf))
* **deps:** update go dependencies ([#63](https://github.com/jp2195/pyre/issues/63)) ([813b27c](https://github.com/jp2195/pyre/commit/813b27cc2e9e43cbbb18b6f709e1a3e8fb0c4df6))
* **deps:** update module charm.land/bubbles/v2 to v2.2.1 ([#64](https://github.com/jp2195/pyre/issues/64)) ([51b024a](https://github.com/jp2195/pyre/commit/51b024affa6545396a97e6d3a0336610edf54edb))
* **deps:** update module charm.land/lipgloss/v2 to v2.0.6 ([#60](https://github.com/jp2195/pyre/issues/60)) ([2195114](https://github.com/jp2195/pyre/commit/2195114cd8a9f3bd05545948261802376f77f6da))
* **logs:** bound the device query bar's width to the terminal ([a32207f](https://github.com/jp2195/pyre/commit/a32207fafd62d8dc6de8d43ac53adaf72336ad77))
* **logs:** don't blame the device query for a local filter's zero rows ([f2e4730](https://github.com/jp2195/pyre/commit/f2e4730c644754ad1ff46b8ef5fffe75d50a0806))
* **logs:** give every log page fetch an identity ([8eba02b](https://github.com/jp2195/pyre/commit/8eba02b3d3a1ebffc7444cfa09ae50b3983065c1))
* **logs:** refetch the visible tab when its page is dropped ([833a1bc](https://github.com/jp2195/pyre/commit/833a1bc4cf4fb8649a540c53498e5bfccb45487a))
* **logs:** stop re-sending a query the tab has already refused ([4bc2f12](https://github.com/jp2195/pyre/commit/4bc2f12ba40bf39288d77214c6f16c03b5a4eb14))
* stop reporting figures the device never gave ([3accd72](https://github.com/jp2195/pyre/commit/3accd72cbcde2860b8917cb2b78c5bbf13dd6c10))
* **tui:** clear cached view data when the target device or firewall changes ([b67e033](https://github.com/jp2195/pyre/commit/b67e033d7996c9441a3f9a4316ee3b59c8e5941f))
* **tui:** drop responses that outlive a connection or target switch ([235e36d](https://github.com/jp2195/pyre/commit/235e36dae31bd191279415bf5e23c2ced000dd0b))
* **tui:** hoist the refresh call out of its return statement ([8dc23f5](https://github.com/jp2195/pyre/commit/8dc23f55b5d08d175f827bb81266bf161b5a273a))
* **tui:** make the login flow answer for itself ([2948a6f](https://github.com/jp2195/pyre/commit/2948a6fbd8fc254318b09d0bfb7616e2e65a5bde))
* **tui:** route pasted text to the focused field ([50ddf78](https://github.com/jp2195/pyre/commit/50ddf781708b9ff134823b569eba2606d8e62afb))
* **tui:** scope log errors per tab, bracket IPv6 hosts, drop mouse mode ([c55f69e](https://github.com/jp2195/pyre/commit/c55f69e132d303d8525f762199a3703afeea55a4))
* **tui:** stop the connection form discarding edits and TLS settings ([4e3c69a](https://github.com/jp2195/pyre/commit/4e3c69a80d4b7409febb9b6626e624b0df5b8bb9))
* **views:** align log table headings with their columns ([ec75b23](https://github.com/jp2195/pyre/commit/ec75b23b356c2aaf58620d0a67dd03472cb8bad2))
* **views:** fit the connection screens to the terminal, and cover them ([4922281](https://github.com/jp2195/pyre/commit/4922281530248a53b8c8eb179335ccb8b86e17bc))
* **views:** frame the routes view, size log tables to the terminal ([0be5f55](https://github.com/jp2195/pyre/commit/0be5f5526ec71b20eccd07998dbe2f244990b222))
* **views:** give data table rows a single left edge ([fed93b3](https://github.com/jp2195/pyre/commit/fed93b362dc114d76d9c48d8cc6cd0f476afaf9d))
* **views:** keep every screen inside the terminal at any width ([17e2b37](https://github.com/jp2195/pyre/commit/17e2b3778b8b70f23d36b2e7047248602ac1d5d3))
* **views:** stop the command palette input overflowing its box ([133337f](https://github.com/jp2195/pyre/commit/133337f53f91e061053d104a029b2619faa16538))
* **views:** wrap text by display width and break unbreakable runs ([50e212c](https://github.com/jp2195/pyre/commit/50e212cad08f8ae5882a0e5521ff12c6b98ffab9))
* **views:** wrap the filter-text lines instead of letting them overflow ([098739b](https://github.com/jp2195/pyre/commit/098739b07d9dcf9dde6ebd99a9e4976cbc0b4b71))


### Performance Improvements

* **api:** cap concurrent requests, identify pyre, honor proxy settings ([0bddf6f](https://github.com/jp2195/pyre/commit/0bddf6f41d354c62234041e143a51effda8126e4))
* **api:** learn the device clock with one probe, not three ([0174e77](https://github.com/jp2195/pyre/commit/0174e77c0be761d0e24b666772c7e4770eba0446))
* **api:** stop re-probing rulebases the device says it does not have ([f6dc972](https://github.com/jp2195/pyre/commit/f6dc9723a4e065a92db7cd188f201338b1af5a12))

## [1.5.4](https://github.com/jp2195/pyre/compare/v1.5.3...v1.5.4) (2026-08-05)


### Bug Fixes

* **tui:** support pasting into text inputs ([#54](https://github.com/jp2195/pyre/issues/54)) ([5601c0a](https://github.com/jp2195/pyre/commit/5601c0aea2e8a3ce2c0e251e1bb636f067abff43))

## [1.5.3](https://github.com/jp2195/pyre/compare/v1.5.2...v1.5.3) (2026-08-04)


### Bug Fixes

* **deps:** update module charm.land/lipgloss/v2 to v2.0.5 ([#45](https://github.com/jp2195/pyre/issues/45)) ([4b6e393](https://github.com/jp2195/pyre/commit/4b6e393cdd2e2259696494cbd2ee2ccb214c381b))
* **deps:** update module go.yaml.in/yaml/v4 to v4.0.0-rc.6 ([#42](https://github.com/jp2195/pyre/issues/42)) ([3dffc59](https://github.com/jp2195/pyre/commit/3dffc59e8993dc5087c2f8f5b964d2fa2f81d2b4))

## [1.5.2](https://github.com/jp2195/pyre/compare/v1.5.1...v1.5.2) (2026-06-13)


### Documentation

* tidy dashboard notes and trigger the v1.5.2 release ([#37](https://github.com/jp2195/pyre/issues/37)) ([36c3033](https://github.com/jp2195/pyre/commit/36c3033e0e38d24f8c14400a902a852e3dfc6482))

## [1.5.1](https://github.com/jp2195/pyre/compare/v1.5.0...v1.5.1) (2026-06-13)


### Bug Fixes

* **release:** emit cosign Sigstore bundle instead of legacy sig/pem ([#34](https://github.com/jp2195/pyre/issues/34)) ([1bc859b](https://github.com/jp2195/pyre/commit/1bc859b5f9b6bc17abb1c7d217b595484c496cea))

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
