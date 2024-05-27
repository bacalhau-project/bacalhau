/*
Package stream provides a NATS client for streaming records between clients with asynchronous
response handling.

Overview:

This package implements a client leveraging NATS for sending and receiving streaming data records.
It abstracts NATS connections, subscriptions, and message handling complexities, offering a
simplified interface for data streaming. The client supports multiplexing multiple streams over a
single NATS subscription, handling responses from different streams using a unique token-based
mechanism. Additionally, the package introduces a Writer component, designed to abstract the
complexities of data encoding and NATS publishing into a simple, intuitive interface.

How It Works:

  - The ConsumerClient part of the package manages dynamic inboxes for each streaming session, facilitating
    the sending of data and listening for responses on dedicated subjects. It leverages the NATS
    publish-subscribe model for asynchronous communication, efficiently routing and correlating
    messages to their respective streams.

  - The ProducerClient component is instrumental in maintaining the smooth exchange of information between the server and client.
    Its primary role is to manage the heartbeat mechanism that plays a pivotal role in synchronizing the client and server.
    A 'heartbeat' is essentially a methodical signal dispatched by the ProducerClient at fixed intervals.
    Each heartbeat carries a list of active stream IDs, which instructs the ConsumerClient on which streams to keep
    open for continued communication.Taking advantage of NATS's asynchronous publish-subscribe model, the ProducerClient ensures
    that heartbeat messages are efficiently directed to the right ConsumerClient. In turn, the ConsumerClient recognizes
    the active streams and keeps them open for receiving subsequent messages. It's important that the ProducerClient
    receives a response to each heartbeat within a certain timeout period. This response comprises a list of non-active stream
    IDs sent by the ConsumerClient. The ProducerClient can interpret this response to understand which streams are unnecessary,
    prompting it to stop publishing to those specific inbox subjects that were initially created by the ConsumerClient.
    In the absence of a timely heartbeat response, the ProducerClient assumes the ConsumerClient no longer requires the information.
    Consequently, it ceases publishing to those specific inbox subjects, thus preserving resources and ensuring efficient communication.
    Through this mechanism, the ProducerClient not only facilitates efficient communication but also effective resource management.
    It stops the overuse of streams that are no longer in demand, thereby maintaining the application's responsiveness and real-time prowess

  - The Writer component allows for easy publishing of structured data to any NATS subject. It
    integrates tightly with the ConsumerClient, utilizing the same connection for streamlined data streaming.
    The Writer simplifies the publication process, automatically handling data serialization and
    supporting graceful stream closure with custom codes.

Multiplexing Streams:

To efficiently handle multiple streams, the client uses a single wildcard subscription for all
responses. Each request is sent with a unique response subject (derived from a base inbox prefix),
with responses routed back to this subject. The client demultiplexes incoming messages by extracting
a token from the response subject, identifying the correct stream (or "bucket") for the message.
This approach allows managing multiple concurrent streams with minimal overhead, leveraging NATS's
lightweight subjects and messaging capabilities.

Key Features:

  - Asynchronous Streaming: Supports asynchronous data streaming, allowing clients to send data and
    receive responses without blocking.

  - Context Support: Integrates with Go's context package for timeouts, cancellation, and deadlines
    for streaming requests.

  - Multiplexing: Efficiently multiplexes multiple streams over a single NATS subscription, using
    unique response subjects for message correlation.

  - Error Handling: Provides robust error handling, including custom error codes for stream-related
    errors (e.g., bad data, normal closure).

  - Data Publication: The Writer simplifies structured data publishing to NATS, with support for
    automatic serialization and stream closure signals.

  - HeartBeating: The producer client in regular interval sends information to consumer client about the
    active stream ids.

Usage:

Initialize the client with a NATS connection and use the provided methods to send streaming requests
and handle responses. The client manages the NATS subscription and response routing, simplifying the
process of working with streaming data.

Example:

	params := stream.ConsumerClientParams{Conn: natsConn}
	client, err := stream.NewConsumerClient(params)
	if err != nil {
	    log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	responseChan, err := client.OpenStream(ctx, "subject", []byte("data"))
	if err != nil {
	    log.Fatal(err)
	}

	for asyncResult := range responseChan {
	    if asyncResult.Err != nil {
	        log.Printf("Received an error: %v", asyncResult.Err)
	        continue
	    }
	    log.Printf("Received data: %s", asyncResult.Value)
	}

This package leverages the concurrency.AsyncResult type for handling asynchronous responses,
allowing clients to distinguish between successful data responses and error conditions.

Note: This client is designed for use with NATS and requires an established NATS connection to
function. It does not handle NATS connection management, which must be performed separately by
the user.
*/
package stream
