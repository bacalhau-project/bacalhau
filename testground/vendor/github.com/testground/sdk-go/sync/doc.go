// The sync package contains the distributed coordination and choreography
// facility of Testground.
//
// The sync service is lightweight, and uses Redis recipes to implement
// coordination primitives like barriers, signalling, and pubsub. Additional
// primitives like locks, semaphores, etc. are in scope, and may be added in the
// future.
//
// Constructing sync.Clients
//
// To use the sync service, test plan writers must create a sync.DefaultClient via the
// sync.NewBoundClient constructor, passing a context that governs the lifetime
// of the sync.DefaultClient, as well as a runtime.RunEnv to bind to. All sync
// operations will be automatically scoped/namespaced to the runtime.RunEnv.
//
// Infrastructure services, such as sidecar instances, can create generic
// sync.Clients via the sync.NewGenericClient constructor. Such clients are not
// bound/constrained to a runtime.RunEnv, and instead are required to pass in
// runtime.RunParams in the context.Context to all operations. See WithRunParams
// for more info.
//
// Recommendations for test plan writers
//
// All constructors and methods on sync.DefaultClient have Must* versions, which panic
// if an error occurs. Using these methods in combination with runtime.Invoke
// is safe, as the runner captures panics and records them as test crashes. The
// resulting code will be less pedantic.
//
// We have added sugar methods that compose basic primitives into frequently
// used katas, such as client.PublishSubscribe, client.SignalAndWait,
// client.PublishAndWait, etc. These katas also have Must* variations. We
// encourage developers to adopt them in order to streamline their code.
//
// Garbage collection
//
// The sync service is decentralised: it has no centralised actor, dispatcher,
// or coordinator that supervises the lifetime of a test. All participants in a
// test hit Redis directly, using its operations to implement the sync
// primitives. As a result, keys from past runs can accumulate.
//
// Sync clients can participate in collaborative garbage collection by enabling
// background GC:
//
//   client.EnableBackgroundGC(ch) // see method godoc for info on ch
//
// GC uses SCAN and OBJECT IDLETIME operations to find keys to purge, and its
// configuration is controlled by the GC* variables.
//
// In the standard testground architecture, only sidecar processes are
// participate in GC:
package sync
