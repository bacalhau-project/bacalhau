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


* Rule: According to the golang docs, context (ctx) should be the first parameter for any function that uses it.
* Generally, add context where possible. However, for things that do not require a context, you can skip using it for cleanliness. For example, if you have a function which provisions a struct, or does other things that we do not expect to be traced, you can skip adding context to it.
  * Realize, of course, that this may come back to bite you if you want to add tracing to that function later.
  * However, if you trace the entire function, and this is a sub function, you will get a trace for the entire function, which may be enough to isolate the problem tracing is designed to provide.