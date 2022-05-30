# CHANGELOG

## [v1.16.6](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.6)

#### Core:
  - docs(readme): update changelogs for v1.16.6 #320 (@javiercri)
  - fix: revert issue #289 #325 (@javiercri)

#### Plugins:
  - chore(plugins) Add webhdfs plugin #321 (@luksan47)

#### Helm:
  - None

#### CI:
  - None

## [v1.16.5](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.5)

#### Core:
  - docs(readme): update changelogs for v1.16.5 #320 (@javiercri)
  - fix: update ruby to 2.7.6 #319 (@slimm609)

#### Plugins:
  - None

#### Helm:
  - None

#### CI:
  - None

## [v1.16.4](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.4)

#### Core:
  - docs(readme): update changelogs for v1.16.4 #312 (@javiercri)
  - fix: update CVEs #317 (@slimm609)

#### Plugins:
  - Cleanup SERVERENGINE_SOCKETMANAGER socket files usually found in /tmp #316 (@danlenar)

#### Helm:
  - None

#### CI:
  - None

## [v1.16.3](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.3)

#### Core:
  - docs(readme): update changelogs for v1.16.3 #312 (@javiercri)
  - chore: add pipeline for release builds (@javiercri)
  - fix: Issue#289 Mounted file issue fix part-2 #306 (@vkadi)
  - fix: Show error when dir is missing #305 (@javiercri)
  - fix: Fix cri-o Config when log line contains a json #301 (@jonas.rutishauser)
  - chore(fluentd): upgrade fluentd v1.14.4 #300 (@Cryptophobia)
  - chore(fluentd): upgrade fluentd and ruby 2.7.5 #297 (@Cryptophobia)

#### Plugins:
  - fix(plugins) re-added fluentd Splunk HEC plugin #307 #309 (@xelalex)

#### Helm:
  - None

#### CI:
  - None

## [v1.16.2](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.2)

#### Core:
  - docs(readme): update changelogs for v1.16.1 #288 (@Cryptophobia)
  - fix: fix datetime parse to extended format #291 (@Cryptophobia)
  - chore: upgrade date gem and bundler #292  (@Cryptophobia)
  - fix: react to pod events to regen preprocess conf #293 (@sabdalla80 @Cryptophobia)

#### Plugins:
  - None

#### Helm:
  - None

#### CI:
  - None

## [v1.16.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.1)

#### Core:
  - docs(readme): update changelogs for v1.16.0 #283 (@Cryptophobia)
  - chore(fluentd): upgrade fluentd to v1.14.2 #286 (@Cryptophobia)
  - chore(docs): update docs for v1.16.1 release #287  (@Cryptophobia)

#### Plugins:
  - None

#### Helm:
  - None

#### CI:
  - None

## [v1.16.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.16.0)

#### Core:
  - docs(readme): update changelogs for v1.15.3 #259 (@Cryptophobia)
  - chore(go): upgrade to go 1.17 #260 (@Cryptophobia)
  - fix(kubectl): add update to kubectl manifests #263 (@Cryptophobia)
  - chore(k8s): regen k8s clients with code-generator #265 (@Cryptophobia)
  - fix(relabel): process timeout_labels also #270 (@Cryptophobia)
  - docs(README): add docs about label rewriting etc. #271 (@Cryptophobia)
  - chore(fluentd): upgrade fluentd v1.14.1 and gems #272 (@Cryptophobia)
  - fix(reloader): revert to reload endpoint and info #273 (@Cryptophobia)
  - fix(reloader): we can do better error logging #275 (@Cryptophobia)
  - ref(datasource): ns discovery on startup #278 (@Cryptophobia)
  - fix(reloader): move back to config.gracefulReload #279 (@Cryptophobia)
  - fix(controller): fix tracking of num config ns #282 (@Cryptophobia)

#### Plugins:
  - fix(plugins): remove splunk plugins for #266 #274 (@Cryptophobia)
  - chore(plugin): use fork of detect-exception gem #280 (@Cryptophobia)
  - fix(plugins): add back splunkhec 2.1 plugin #281 (@Cryptophobia)

#### Helm:
  - None

#### CI:
  - None

## [v1.15.3](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.15.3)

#### Core:
  -  chore(fluentd): upgrade fluentd v1.14.0 and gems #250 (@Cryptophobia)
  -  chore(go.mod): upgrade k8s.io packages to v0.21.4 #248 (@Cryptophobia)
  -  chore(ruby): upgrade to ruby version 2.7.4 #251 (@Cryptophobia)
  -  docs(readme): update changelogs for v1.15.2 #242 (@Cryptophobia)
  -  docs(repo): missing namespace in annotate cmd #244 (@brunowego)
  -  chore(gems): upgrade json to 2.5.1 #254 (@Cryptophobia)
  -  feat(context): propagate context from main func #256 (@Cryptophobia)

#### Plugins:
  -  chore(plugins): upgrade fluentd plugin gems #252 (@Cryptophobia)
  -  chore(conf): upgrade systemd fluentd configs #257 (@Cryptophobia)
  -  chore(plugins): upgrade kubernetes metadata plugin #258 (@Cryptophobia)

#### Helm:
  - None

#### CI:
  - None

## [v1.15.2](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.15.2)

#### Core:
  - docs(readme): update changelogs for v1.15.1 #229 (@Cryptophobia)
  - fix(reloader): use update for status annotation #234 (@Cryptophobia)
  - fix(reloader): remove the IsKnowFromBefore logic #235 (@Cryptophobia)
  - fix(kube): ignore errors.IsConflict on update call #237 (@Cryptophobia)

#### CI:
  - None

#### Helm:
  - feat(charts): add update on ns role and template for rbac api #238 (@Cryptophobia)
  - feat(charts): bump up chart version to 0.4.0 #240 (@Cryptophobia)

#### Plugins:

  - chore(deps): Bump addressable from 2.7.0 to 2.8.0 in /base-image #230 (@dependabot)
  - chore(fluentd): upgrade fluentd v1.13.2 and gems #231 (@Cryptophobia)
  - chore(fluentd): upgrade fluentd v1.13.3 and gems #236 (@Cryptophobia)

## [v1.15.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.15.1)

#### Core:
  - docs(readme): update changelogs for v1.15.0 #219 (@Cryptophobia)
  - fix(kubernetes.conf): set read_from_head to false #220 (@Cryptophobia)
  - Revert "fix(kubernetes.conf): set read_from_head to false" #223 (@Cryptophobia)
  - chore(deps): various updates and add jemalloc lib #224 (@Cryptophobia)
  - fix(generator): fix status annot on unused ns #227 (@Cryptophobia)
  - fix(Dockerfile): add tdnf clean commands #228 (@Cryptophobia)

#### CI:
  - None

#### Helm:
  - None

#### Plugins:
  - chore(fluentd): upgrade fluentd v1.13.1 and gems #225 (@Cryptophobia)
  - fix(plugins): fix for fluent-plugin-loggly gem #226 (@Cryptophobia)

## [v1.15.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.15.0)

#### Core:
  - docs(readme): update changelogs for v1.14.1 #210 (@Cryptophobia)
  - chore(deps): Bump rexml from 3.1.9 to 3.2.5 in /base-image/basegems #212 (@dependabot)
  - chore(fluentd): upgrade fluentd to v1.12.4 and bundler to >= 2.2.18 #215 (@Cryptophobia)
  - chore(fluentd): upgrade fluentd v1.13.0 and gems #216 (@Cryptophobia)
  - docs(readme): update changelogs for v1.15.0 #217 (@Cryptophobia)
  - fix(gems): peg kubeclient version in Gemfile #218 (@Cryptophobia)

#### CI:
  - None

#### Helm:
  - None

#### Plugins:
  - chore(fluentd): upgrade fluentd v1.13.0 and gems #216 (@Cryptophobia)

## [v1.14.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.14.1)

#### Core:
  - chore(docs): add better build instructions #200 (@Cryptophobia)
  - fix(reloader): matadata/labels key can have a period . #201 (@Cryptophobia)
  - fix(base-image): add findutils for validate script #202 (@Cryptophobia)
  - fix(reloader): fix ctx.Namespace typo #204 (@Cryptophobia)
  - feat(reloader): add custom --buffer-mount-folder option #206 (@Cryptophobia)
  - fix(config.go): fix buffer path name check #207 (@Cryptophobia)
  - feat(fluentd): upgrade fluentd to v1.12.3 #208 (@Cryptophobia)

#### CI:
  - None

#### Helm:
  - Fix typo in values.yaml #205 (@Revellski)

#### Plugins:
  - chore(plugins): use loginsight gem 1.0.0 #203 (@Cryptophobia)

## [v1.14.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.14.0)

#### Core:
  - Make fluentd log level configurable (#196) (@kkapoor1987)
  - feat(kubectl): add manifests for kubectl install (#192) (@Cryptophobia)
  - feat(fluentd): upgrade fluentd to v1.12.2 (#193) (@Cryptophobia)
  - feat(reloader): sorts configs so they are ordered (#194) (@Cryptophobia)
  - chore(base-image): upgrade gems and bundler (#195) (@Cryptophobia)
       - upgrade ruby to 2.7.3
       - upgrade bundler to 2.2.15
  - chore(go): upgrade to go1.16 and add linting (#197) (@Cryptophobia)

#### CI:
  - chore(go): upgrade to go1.16 and add linting (#197) (@Cryptophobia)

#### Helm:
  - Make fluentd log level configurable (#196) (@kkapoor1987)

## [v1.14.0-beta.3](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.14.0-beta.3)

#### Core:
  - Move to graceful reload endpoint and error (#189) (@slimm609)
  - fix(config-reloader): add timeout seconds on startup (#190) (@Cryptophobia)
  - chore(base-image): upgrade to use photon:4.0 (#186) (@Cryptophobia)

#### Plugins:
  - chore: upgrade vmware-log-intelligence to 2.0.6 (#187) (@Cryptophobia)

## [v1.14.0-beta.2](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.14.0-beta.2)

#### Core:
  - Migrate to go modules (#152) (@viveksyngh)
  - Add timeout and log for validation (#180) (@viveksyngh)

#### CI:
  - Update kind github action to fix ci (#152) (@viveksyngh)
  - Remove travis ci config (#169) (@viveksyngh)

#### Helm:
  - Fix helm chart deprecated endpoints (#168) (@slimm609)
  - Update helm to helm3 in ci (#170) (@viveksyngh)
  - Adds support for SA annotations (#167) (@kkapoor1987)

#### Plugins:
  - Update ruby, fluentd, and gem plugins (#173) (@Cryptophobia)
      - Update fluentd gem to 1.11.2
      - Update ruby version to 2.7.2 (webrick CVE)
      - **DEPRECATION**: removing fluent-plugin-scribe as no longer maintained
  - Get vmware-log-intelligence from rubygems (#174) (@Cryptophobia)
      - Update fluent-plugin-vmware-log-intelligence to 2.0.5
  - Feat(splunk-hec): add plugin & fix dependencies (#179) (@jliao2011)
      - fluent-plugin-splunk-hec is added

## [1.14.0-beta.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.14.0-beta.1)

* Core: Expose config-reloader prometheus metrics (#147) (@tommasopozzetti )
* CI: Update github action to include kind test (#148) (@viveksyngh)
* Plugin: Update log intelligence plugin to 2.0.4 (#150) (@viveksyngh)

## [1.13.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.13.0)

* Core: Introduce 'crd' datasource (#111) (@tommasopozzetti )
* Helm: improve Helm chart and bump version to 0.3.4 (#130) (@dimalbaby )
* Core: Make the admin namespace configurable (#118 )(@tommasopozzetti )
* Helm: PriorityClassName and add ServiceMonitor in helm chart (#102)(@evilrussian )
* Image: Reduce image size (@dimalbaby )
* Core: Update CI github workflows (#129) (@tsunny )
* Plugin: Upgrade fluent-plugin-azure-loganalytics plugin (#136) (@floriankoch )

## [1.13.0-beta.2](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.13.0-beta.2)

* Helm: improve Helm chart and bump version to 0.3.4 (#130) (@dimalbaby )

## [1.13.0-beta.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.13.0-beta.1)

* Core: Introduce 'crd' datasource (#111) (@tommasopozzetti )
* Core: Make the admin namespace configurable (#118 )(@tommasopozzetti )
* Helm: PriorityClassName and add ServiceMonitor in helm chart (#102)(@evilrussian)
* Image: Reduce image size

## [1.12.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.12.0)

* Build kube-fluentd-operator on photon (#115) (@dimalbaby)

* Plugins: Add `fluent-plugin-vmware-loginsight` (@naveens)

* Plugins: Add `fluent-plugin-grafana-loki` (@SerialVelocity)

* Core: Cleanup unused namespace configuration files (#110) (@tommasopozzetti)

* Core: Fix labels not being rewritten in copy plugin (#86) (@tommasopozzetti)

* Core: Fix $label processor ignoring errors and not accepting "/" (#93) (@tommasopozzetti)

* Core: Fix namespace fluentd status annotation update (#92) (@tommasopozzetti)

* Core: Fix slow CRI-O log parsing (#91) (@jonasrutishauser)

* Core: Introduce a more efficient event-triggered reload system (#89)  (@tommasopozzetti)

* Core: Add new processor to allow controlled tag rewriting (#88) (@tommasopozzetti)

* Helm: improve Helm chart and bump version to 0.3.3 (#95) (@cablespaghetti)

* Fluentd: upgrade fluentd to 1.9.1

## [1.11.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.11.0)

* Plugins: Add `fluent-plugin-verticajson` again (@skalickym)

* Plugins: Add `fluent-plugin-uri-parser` (@rjackson90)

* Plugins: Add `fluent-plugin-out-http`

* Plugins: update to latest compatible versions

* Core: Efficient Kubernetes API Updates using Informers (#75) (@rjackson90)

* Fluend: Add support for CRI-O format (#80) (@jonasrutishauser)

## [1.10.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.10.0)

* Fluentd: Upgrade to 1.5.2

* Fluentd: Use libjemalloc from base image improving multi-platform build (@james-crowley)

* Plugins: Add `fluent-plugin-datadog` (@chirauki)

* Plugins: Add `fluent-plugin-multi-format-parser`

* Plugins: Add `fluent-plugin-json-in-json-2`

* Plugins: Add `fluent-plugin-gelf-hs`

* Plugins: Add `fluent-plugin-azure-loganalytics` (@zachomedia)

* Plugins: Upgrade LINT plugin to version 2.0.0

* Plugins: Remove vertica plugins as they fail to build with 1.5.2

* Core: `<plugin>`'s content is expanded correctly (#62) (@)

* Core: Allow the use of virtual `<plugin>` in store directive (#71) (@zachomedia)

* Core: Add subPath support to mounted-file (#63) (@jonasrutishauser)

## [1.9.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.9.0)

* Fluentd: BREAKING CHANGE: the plugin `fluent-plugin-parser` is not installed anymore. Instead the built-in `@type parser`
  is used and it has a different syntax (https://github.com/tagomoris/fluent-plugin-parser)

* Fluentd: Updated K8S integration to latest recommended for Fluentd 1.2.x

* Plugins: refresh all plugins to latest versions as of the release date.

* Plugins: add plugin `cloudwatch-logs` (@conradj87)

* Helm: improve Helm chart and bump version to 0.3.1 (@PabloCastellano)

## [1.8.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.8.0)

* Fluentd: replace fluent-plugin-amqp plugin with fluent-plugin-amqp2

* Fluentd: update fluentd to 1.2.6

* Fluentd: add plugin `fluent-plugin-vertica` to base image (@rhatlapa)

* Core configmaps per namespace support (#31) (@coufalja)

* Helm: chart updated to 0.3.0 adding support for multiple configmaps per namespace

## [1.7.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.7.0)

* Fluentd: add plugin `extract` to base image - a simple filter plugin to enrich log events based on regex and templates

* Fluentd: add plugin `fluent-plugin-amqp2` to base image (@CoufalJa)

* Fluentd: add plugin `fluent-plugin-grok-parser` to base image (@CoufalJa)

* Core: handle gracefully a missing `kubernetes.labels` field (#23)

* Core: Add `strict true|false` option to the logfmt parser plugin (#27)

* Core: Add `add_labels` to  `@type mounted_file` plug-in (#26)

* Core: Fix `@type mounted_file` for multi-container pods producing log files (#29)

* Helm: `podAnnotations` lets you annotate the daemonset pods (@cw-sakamoto) (#25)

## [1.6.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.6.0)

* Core: plugin macros (#19). This feature lets you reuse output plugin definitions.

* Fluentd: add `fluent-plugin-kafka` to base image (@dimitrovvlado)

* Fluentd: add `fluent-plugin-mail` to base image

* Fluentd: add `fluent-plugin-mongo` to base image

* Fluentd: add `fluent-plugin-scribe` to base image

* Fluentd: add `fluent-plugin-sumologic_output` to base image

## [1.5.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.5.0)

* Core: extreme validation (#15): namespace configs are run in sandbox to catch more errors

* Helm: export a K8S\_NODE\_NAME var to the fluentd container (@databus23)

* Helm: `extraEnv` lets you inject env vars into fluentd

## [1.4.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.4.0)

* Fluentd: add `fluent-plugin-kinesis` to base image (@anton-107)

## [1.3.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.3.0)

* Fluentd: add `fluent-plugin-detect-exceptions` to base image

* Fluentd: add `fluent-plugin-out-http-ext` to base image

* Core: add plugin `truncating_remote_syslog` which truncates the tag to 32 bytes as per RFC 5424

* Core: support transparent multi-line stacktrace collapsing

## [1.2.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.2.0)

* Core: Share log streams between namespaces

* Helm: Mount secrets/configmaps (tls mostly) ala elasticsearch (@sneko)

* Fluentd: Update base-image to fluentd-1.1.3-debian

* Fluentd: Include Splunk plugin into base-image (@mhulscher)

* Fix(Helm): properly set the `resources` field of the reloader container. Setting them had no effect until now (@sneko)

## [1.1.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.1.0)

* Core: ingest log files from a container filesystem

* Core: limit impact of log-router to a set of namespaces using `--namespaces`

* Helm: add new property `kubeletRoot`

* Helm: add new property `namespaces[]`:

## [1.0.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.0.1)

* Fluentd: install plugin `fluent-plugin-concat` in Fluentd

* Core: support for default configmap name using `--default-configmap`

## [1.0.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.0.0)

* Initial version
