module github.com/bacalhau-project/bacalhau

go 1.21

toolchain go1.21.11

require (
	github.com/BTBurke/k8sresource v1.2.0
	github.com/Masterminds/semver v1.5.0
	github.com/aws/aws-sdk-go-v2 v1.25.1
	github.com/aws/aws-sdk-go-v2/config v1.27.3
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.16.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.51.0
	github.com/aws/smithy-go v1.20.1
	github.com/bmatcuk/doublestar/v4 v4.6.1
	github.com/c2h5oh/datasize v0.0.0-20220606134207-859f65c6625b
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/docker/docker v25.0.4+incompatible
	github.com/dylibso/observe-sdk/go v0.0.0-20231201014635-141351c24659
	github.com/fatih/structs v1.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-playground/validator/v10 v10.16.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.1
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/imdario/mergo v0.3.16
	github.com/ipfs/boxo v0.18.0
	github.com/ipfs/go-cid v0.4.1
	github.com/ipfs/go-ipld-format v0.6.0
	github.com/ipfs/go-unixfsnode v1.9.0
	github.com/ipfs/kubo v0.27.0
	github.com/ipld/go-car/v2 v2.13.1
	github.com/ipld/go-codec-dagpb v1.6.0
	github.com/ipld/go-ipld-prime v0.21.0
	github.com/jedib0t/go-pretty/v6 v6.5.3
	github.com/joho/godotenv v1.5.1
	github.com/labstack/echo/v4 v4.11.4
	github.com/lestrrat-go/jwx v1.2.29
	github.com/libp2p/go-libp2p v0.33.0
	github.com/mattn/go-isatty v0.0.20
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/multiformats/go-multiaddr v0.12.2
	github.com/nats-io/nats-server/v2 v2.10.11
	github.com/nats-io/nats.go v1.34.0
	github.com/nats-io/nuid v1.0.1
	github.com/open-policy-agent/opa v0.60.0
	github.com/opencontainers/image-spec v1.1.0-rc5
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58
	github.com/pkg/errors v0.9.1
	github.com/ricochet2200/go-disk-usage/du v0.0.0-20210707232629-ac9918953285
	github.com/rs/zerolog v1.31.0
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	github.com/swaggo/swag v1.16.2
	github.com/tetratelabs/wazero v1.6.0
	github.com/theckman/yacspin v0.13.12
	github.com/vincent-petithory/dataurl v1.0.0
	go.etcd.io/bbolt v1.3.8
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.40.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.26.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.22.0
	go.opentelemetry.io/otel/metric v1.26.0
	go.opentelemetry.io/otel/sdk v1.26.0
	go.opentelemetry.io/otel/sdk/metric v1.24.0
	go.opentelemetry.io/otel/trace v1.26.0
	go.ptx.dk/multierrgroup v0.0.3
	go.uber.org/mock v0.4.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.21.0
	golang.org/x/exp v0.0.0-20240213143201-ec583247a57a
	gopkg.in/alessio/shellescape.v1 v1.0.0-20170105083845-52074bc9df61
	k8s.io/apimachinery v0.29.0
	k8s.io/kubectl v0.29.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.23.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/swag v0.22.7 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190812055157-5d271430af9f // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20230524184225-eabc099b10ab // indirect
	github.com/ipfs/go-log/v2 v2.5.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/multiformats/go-multicodec v0.9.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nats-io/jwt/v2 v2.5.3 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.15.0 // indirect
	gonum.org/v1/gonum v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240311132316-a219d84964c2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240314234333-6e1732d8331c // indirect
	k8s.io/cli-runtime v0.29.0 // indirect
)

require (
	bazil.org/fuse v0.0.0-20200407214033-5883e5a4b512 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/benbjohnson/clock v1.3.5
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/crackcomm/go-gitignore v0.0.0-20231225121904-e25f5bc08668 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1
	github.com/facebookgo/atomicfile v0.0.0-20151019160806-2de1f203e7d5 // indirect
	github.com/fatih/color v1.16.0
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ipfs/bbloom v0.0.4 // indirect
	github.com/ipfs/go-bitfield v1.1.0 // indirect
	github.com/ipfs/go-block-format v0.2.0 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/ipfs/go-ds-measure v0.2.0 // indirect
	github.com/ipfs/go-fs-lock v0.0.7 // indirect
	github.com/ipfs/go-ipfs-cmds v0.10.0 // indirect
	github.com/ipfs/go-ipfs-util v0.0.3 // indirect
	github.com/ipfs/go-ipld-cbor v0.1.0 // indirect
	github.com/ipfs/go-ipld-legacy v0.2.1 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-metrics-interface v0.0.1 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.4.1 // indirect
	github.com/libp2p/go-libp2p-kad-dht v0.24.4 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.6.3 // indirect
	github.com/libp2p/go-libp2p-record v0.2.0 // indirect
	github.com/libp2p/go-libp2p-routing-helpers v0.7.3 // indirect
	github.com/libp2p/go-msgio v0.3.0 // indirect
	github.com/libp2p/go-netroute v0.2.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multistream v0.5.0 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/petar/GoLLRB v0.0.0-20210522233825-ae3b015fd3e9 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/polydawn/refmt v0.89.0 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.47.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/samber/lo v1.39.0
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/cbor v0.0.0-20171005072247-63513f603b11 // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20240109153615-66e95c3e8a87 // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.40.0 // indirect
	go.opentelemetry.io/proto/otlp v1.2.0 // indirect
	go.uber.org/atomic v1.11.0
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.18.0 // indirect
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/term v0.18.0
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0
	golang.org/x/tools v0.18.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/grpc v1.63.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools/v3 v3.3.0 // indirect
	k8s.io/api v0.29.0 // indirect
	k8s.io/client-go v0.29.0 // indirect
	k8s.io/klog/v2 v2.110.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	lukechampine.com/blake3 v1.2.1 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)
