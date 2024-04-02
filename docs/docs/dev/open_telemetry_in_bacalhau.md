# Open Telemetry in Bacalhau


## Background
After a discussion about this with Honeycomb - https://honeycombpollinators.slack.com/archives/CNQ943Q75/p1661657055345219

In outline form:
* Start a new trace for each significant process
  * Trace should span across CLI to Server and back
  * Trace should contain baggage about the trace (e.g. job id, user id, etc)
* New span for every short lived action (e.g. < 10 min)
* New trace for jobs longer than 1 hour
* Try very hard to break up traces into smaller pieces
* In tail sampling: the decision to keep or drop information is done at the traceid level.
  * There's usually a pretty short timeout for decision-time so if something happens 30 minutes into the trace that means you'd want to keep it (errors), the decision to drop it may have been made 28 minutes ago.
  * For this reason I recommend keeping traceid to each job executed in an async queue.
* You can also add your own custom identifiers to things for aggregation sake later.
  * This is a good use for a lot of activities that are enqueued at the same time or otherwise relate to each other.
* You can ALSO have layers of tracing.
  * One for the scheduler and one for the execution.
  * The scheduler would have spans for "adding to queue" and "pulling from queue" so you can measure lag time there.
  * Then each job execution gets its own tracer to attach spans to.
  * You'd basically not sample the scheduler traces and just manage events via more or less detail.
* Give it another identifier for the week-long activity and then add it to spans and traces for every event related to that super-lengthy activity.
* Start a tracer for the server process that makes a new trace each time a request comes from the CLI. If the daemon does other stuff on a schedule or timer then I'd start a new trace for each of those activities.
  * The long running job should be a single trace unless it goes over an hour or.
  * Then the options are to break it after some duration and start a new traceid or figure out the next decomposition of the long running job. Is it computing the whole time? Is it waiting a lot?
  * And really, if you start with a huge trace and see that it makes the waterfall look bad since there's a time limit or span limit, then split it up.
  * If you start with tons of tiny traces and the aggregates don't make sense due to sampling, then maybe combining them back together would help.
* The things we see are that sampling is tougher and waterfalls are wonky if traces are too big and aggregates look weird if traces are broken into too small of a unit.
    * The "make a new tracer" and have parallel output isn't something I'd start with. Only escalate to if the scheduler and jobs are misbehaving and it's unclear why
* The imported "trace" library has all the mechanisms for interacting with the tracer...
  * This example uses a named tracer. https://github.com/open-telemetry/opentelemetry-go/blob/main/example/namedtracer/main.go
* The two things you need in any given place are the otel trace import and the context.
  * You can get the current span from context and either add a child span or add attributes depending on what you're doing.
  * Interacting with the tracer is usually just done at initialization for each service.
* The http/grpc service autoinstrumentation should receive the spancontext from the client and continue to add child spans to that.
  * You still need to configure the exporter on the server since that's not propagated.
  * The trace.start function or withspan block are good ways to make a new span in the current trace.

Some more docs - https://www.honeycomb.io/blog/ask-miss-o11y-opentelemetry-baggage/

https://github.com/honeycombio/example-greeting-service/tree/main/golang


## Tracing in Bacalhau

### Starting a new command
For any top level function (e.g. that could be executed by the CLI), include the following code:

```golang
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), NAME_OF_FUNCTION)
		defer rootSpan.End()
		cm.RegisterCallback(telemetry.Cleanup)
```

where `NAME_OF_FUNCTION` is of the form `folder/file/command` -> `cmd/bacalhau/describe`.

This initiates the cleanup manager, pulls in the cmd Context (which is created in `root.go`).

Then it creates a root span, which is a function that automatically adds helpful things like the environment something is running in, and can be extended in the future.

We then assign the defer to end the span, and register the cleanup manager for shutting down the trace provider.

### Starting a new function
When you start a new function, simply add:
```golang
	ctx, span := system.GetTracer().Start(ctx, NAME_OF_SPAN)
	defer span.End()
```
Here, `NAME_OF_SPAN` should be of the form `toplevelfolder/packagename.functionname` E.g `pkg/computenode.subscriptionEventBidRejected`

The `ctx` variable should come from the function declaration, and if it does not have it, we should see if it makes sense to thread it through from the original call. Reminder, `ctx` should be the first parameter for all functions that require it, according to the Go docs. Please avoid using `context.Background` or otherwise creating a new context. This will mess up tying spans together.

If you do feel the need to create a new one, use the anchor tag (in comments) `ANCHOR`, so that we can search for it.

Additionally, if you would like to add baggage to the span, which must be done for each span created, you can pull it from the context (if it exists). You can do so with the following commands:
```golang
	system.AddNodeIDFromBaggageToSpan(ctx, span)
	system.AddJobIDFromBaggageToSpan(ctx, span)
```

We do check to make sure the baggage already exists and if it doesn't we do not add it. If you attempt to add a baggage that does not exist, we print out the stack trace (but only in debug mode).

You MUST manually add the baggage to the span in the function where you create the new span you create. Attributes do NOT carry through from parent to children spans (though, interestingly, baggage DOES carry through).

If you are adding baggage TO a span, because you're creating a node ID or job ID for example, you can use the following:

```golang
	ctx = system.AddNodeIDToBaggage(ctx, n.ID)
```

This context now carries the baggage forward to any function that references it.

### Philosophy of Logging
Generally, add context and tracing where possible. However, for things that are short and do not perform significant compute, I/O, networking, etc, you can skip context and tracing for cleanliness. For example, if you have a function which provisions a struct, or does other things that we do not expect to be traced, you can skip adding context or tracing to it.

If you trace an entire function, and the thing you are debating to add a trace to is a sub function, you may not need to create a subspan. Generally, if you can imagine any situation in which you would debug a problem in a function, you probably want to add a trace.

Further, you may want to create spans inside functions to trace particular blocks of code. This is not recommended, because it makes using `defer` a challenge, and `defer` gives you a number of nice clean up features that you will want for tracing. A good rule of thumb is if you have something that is long enough to be a span, it should be a function.


Some good reading:
 - https://github.com/honeycombio/honeycomb-opentelemetry-go
 - https://github.com/honeycombio/example-greeting-service/blob/main/golang/year-service/main.go
