module github.com/bacalhau-project/bacalhau

go 1.20

require (
	github.com/BTBurke/k8sresource v1.2.0
	github.com/Masterminds/semver v1.5.0
	github.com/XSAM/otelsql v0.20.0
	github.com/antihax/optional v1.0.0
	github.com/application-research/estuary-clients/go v0.0.0-20221129102826-8a9f3452ad5a
	github.com/aws/aws-sdk-go-v2 v1.18.0
	github.com/aws/aws-sdk-go-v2/config v1.18.19
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.60
	github.com/aws/aws-sdk-go-v2/service/s3 v1.31.1
	github.com/aws/smithy-go v1.13.5
	github.com/bacalhau-project/golang-mutex-tracer v0.0.0-20230214151516-bb996d6e8b46
	github.com/bmatcuk/doublestar/v4 v4.6.0
	github.com/c2h5oh/datasize v0.0.0-20220606134207-859f65c6625b
	github.com/davecgh/go-spew v1.1.1
	github.com/didip/tollbooth/v7 v7.0.1
	github.com/docker/docker v23.0.3+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/fatih/structs v1.1.0
	github.com/felixge/httpsnoop v1.0.3
	github.com/filecoin-project/go-address v1.1.0
	github.com/filecoin-project/go-jsonrpc v0.2.1
	github.com/filecoin-project/go-state-types v0.11.1
	github.com/go-test/deep v1.1.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.7.2
	github.com/imdario/mergo v0.3.15
	github.com/invopop/jsonschema v0.7.0
	github.com/ipfs/go-cid v0.4.1
	github.com/ipfs/go-ipfs-http-client v0.5.0
	github.com/ipfs/go-ipld-format v0.4.0
	github.com/ipfs/go-libipfs v0.6.2
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/ipfs/go-merkledag v0.10.0
	github.com/ipfs/go-unixfs v0.4.4
	github.com/ipfs/go-unixfsnode v1.7.0
	github.com/ipfs/interface-go-ipfs-core v0.11.1
	github.com/ipfs/kubo v0.19.0
	github.com/ipld/go-car/v2 v2.8.0
	github.com/ipld/go-codec-dagpb v1.6.0
	github.com/ipld/go-ipld-prime v0.20.0
	github.com/jedib0t/go-pretty/v6 v6.4.4
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/libp2p/go-libp2p v0.27.4
	github.com/libp2p/go-libp2p-pubsub v0.9.3
	github.com/mattn/go-isatty v0.0.18
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/multiformats/go-multiaddr v0.9.0
	github.com/multiformats/go-multicodec v0.8.1
	github.com/multiformats/go-multihash v0.2.2
	github.com/opencontainers/image-spec v1.0.2
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58
	github.com/pelletier/go-toml/v2 v2.0.7
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5
	github.com/pkg/errors v0.9.1
	github.com/ricochet2200/go-disk-usage/du v0.0.0-20210707232629-ac9918953285
	github.com/rs/zerolog v1.29.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.2
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.1
	github.com/tetratelabs/wazero v1.0.1
	github.com/theckman/yacspin v0.13.12
	github.com/tidwall/sjson v1.2.5
	github.com/vincent-petithory/dataurl v1.0.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.39.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.13.0
	go.opentelemetry.io/otel/metric v0.37.0
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/sdk/metric v0.37.0
	go.opentelemetry.io/otel/trace v1.14.0
	go.ptx.dk/multierrgroup v0.0.2
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.9.0
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	golang.org/x/mod v0.10.0
	k8s.io/apimachinery v0.27.2
	k8s.io/kubectl v0.27.0
	modernc.org/sqlite v1.22.1
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/ProtonMail/go-crypto v0.0.0-20230518184743-7afd39499903 // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.18 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.25 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.25 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.7 // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.4.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/skeema/knownhosts v1.1.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

require (
	bazil.org/fuse v0.0.0-20200407214033-5883e5a4b512 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.0 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/alexbrainman/goissue34681 v0.0.0-20191006012335-3fc7a47baff5 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/ceramicnetwork/go-dag-jose v0.1.0 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/cheggaaa/pb v1.0.29 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/crackcomm/go-gitignore v0.0.0-20170627025303-887ab5e44cc3 // indirect
	github.com/cskr/pubsub v1.0.2 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.0.2 // indirect
	github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/gosigar v0.14.2 // indirect
	github.com/elgris/jsondiff v0.0.0-20160530203242-765b5c24c302 // indirect
	github.com/facebookgo/atomicfile v0.0.0-20151019160806-2de1f203e7d5 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/filecoin-project/go-amt-ipld/v4 v4.0.0 // indirect
	github.com/filecoin-project/go-bitfield v0.2.4 // indirect
	github.com/filecoin-project/go-hamt-ipld/v3 v3.1.0 // indirect
	github.com/flynn/noise v1.0.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.1 // indirect
	github.com/go-git/go-git/v5 v5.7.0
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-pkgz/expirable-cache v1.0.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/google/pprof v0.0.0-20230405160723-4a4c7d95572b // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/hannahhoward/go-pubsub v0.0.0-20200423002714-8d62886cc36e // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huin/goupnp v1.1.0 // indirect
	github.com/iancoleman/orderedmap v0.0.0-20190318233801-ac98e3ecb4b0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ipfs/bbloom v0.0.4 // indirect
	github.com/ipfs/go-bitfield v1.1.0 // indirect
	github.com/ipfs/go-block-format v0.1.1 // indirect
	github.com/ipfs/go-blockservice v0.5.0 // indirect
	github.com/ipfs/go-cidutil v0.1.0 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/ipfs/go-delegated-routing v0.7.0 // indirect
	github.com/ipfs/go-ds-badger v0.3.0 // indirect
	github.com/ipfs/go-ds-flatfs v0.5.1 // indirect
	github.com/ipfs/go-ds-leveldb v0.5.0 // indirect
	github.com/ipfs/go-ds-measure v0.2.0 // indirect
	github.com/ipfs/go-fetcher v1.6.1 // indirect
	github.com/ipfs/go-filestore v1.2.0 // indirect
	github.com/ipfs/go-fs-lock v0.0.7 // indirect
	github.com/ipfs/go-graphsync v0.14.1 // indirect
	github.com/ipfs/go-ipfs-blockstore v1.2.0 // indirect
	github.com/ipfs/go-ipfs-chunker v0.0.5 // indirect
	github.com/ipfs/go-ipfs-cmds v0.8.2 // indirect
	github.com/ipfs/go-ipfs-delay v0.0.1 // indirect
	github.com/ipfs/go-ipfs-ds-help v1.1.0 // indirect
	github.com/ipfs/go-ipfs-exchange-interface v0.2.0 // indirect
	github.com/ipfs/go-ipfs-exchange-offline v0.3.0 // indirect
	github.com/ipfs/go-ipfs-keystore v0.1.0 // indirect
	github.com/ipfs/go-ipfs-pinner v0.3.0 // indirect
	github.com/ipfs/go-ipfs-posinfo v0.0.1 // indirect
	github.com/ipfs/go-ipfs-pq v0.0.3 // indirect
	github.com/ipfs/go-ipfs-provider v0.8.1 // indirect
	github.com/ipfs/go-ipfs-redirects-file v0.1.1 // indirect
	github.com/ipfs/go-ipfs-routing v0.3.0 // indirect
	github.com/ipfs/go-ipfs-util v0.0.2 // indirect
	github.com/ipfs/go-ipld-cbor v0.0.6 // indirect
	github.com/ipfs/go-ipld-git v0.1.1 // indirect
	github.com/ipfs/go-ipld-legacy v0.1.1 // indirect
	github.com/ipfs/go-ipns v0.3.0 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-metrics-interface v0.0.1 // indirect
	github.com/ipfs/go-mfs v0.2.1 // indirect
	github.com/ipfs/go-namesys v0.7.0 // indirect
	github.com/ipfs/go-path v0.3.1 // indirect
	github.com/ipfs/go-peertaskqueue v0.8.1 // indirect
	github.com/ipfs/go-pinning-service-http-client v0.1.2 // indirect
	github.com/ipfs/go-verifcid v0.0.2 // indirect
	github.com/ipld/edelweiss v0.2.0 // indirect
	github.com/ipld/go-car v0.5.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.16.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/koron/go-ssdp v0.0.4 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-doh-resolver v0.4.0 // indirect
	github.com/libp2p/go-flow-metrics v0.1.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.3.0 // indirect
	github.com/libp2p/go-libp2p-gostream v0.5.0 // indirect
	github.com/libp2p/go-libp2p-http v0.4.0 // indirect
	github.com/libp2p/go-libp2p-kad-dht v0.21.1 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.5.0 // indirect
	github.com/libp2p/go-libp2p-pubsub-router v0.6.0 // indirect
	github.com/libp2p/go-libp2p-record v0.2.0 // indirect
	github.com/libp2p/go-libp2p-routing-helpers v0.6.1 // indirect
	github.com/libp2p/go-libp2p-xor v0.1.0 // indirect
	github.com/libp2p/go-mplex v0.7.0 // indirect
	github.com/libp2p/go-msgio v0.3.0 // indirect
	github.com/libp2p/go-nat v0.1.0 // indirect
	github.com/libp2p/go-netroute v0.2.1 // indirect
	github.com/libp2p/go-reuseport v0.2.0 // indirect
	github.com/libp2p/go-yamux/v4 v4.0.0 // indirect
	github.com/libp2p/zeroconf/v2 v2.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.53 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multistream v0.4.1 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/onsi/ginkgo/v2 v2.9.2 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.0 // indirect
	github.com/petar/GoLLRB v0.0.0-20210522233825-ae3b015fd3e9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/polydawn/refmt v0.89.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.2 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.2 // indirect
	github.com/quic-go/quic-go v0.33.0 // indirect
	github.com/quic-go/webtransport-go v0.5.2 // indirect
	github.com/raulk/go-watchdog v1.3.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/samber/lo v1.38.1
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.4 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tidwall/gjson v1.14.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/ucarion/urlpath v0.0.0-20200424170820-7ccc79b76bbb // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/cbor v0.0.0-20171005072247-63513f603b11 // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20230126041949-52956bd4c9aa // indirect
	github.com/whyrusleeping/chunker v0.0.0-20181014151217-fe64bd25879f // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-sysinfo v0.0.0-20190219211824-4a357d4b90b1 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.7.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.12.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.7.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/dig v1.16.1 // indirect
	go.uber.org/fx v1.19.2 // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.53.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.3.0 // indirect
	k8s.io/api v0.27.0 // indirect
	k8s.io/client-go v0.27.0 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
