import 'package:swagger/api.dart';
import 'package:test/test.dart';


/// tests for MiscApi
void main() {
  var instance = new MiscApi();

  group('tests for MiscApi', () {
    // Returns the id of the host node.
    //
    //Future<String> apiServerId() async
    test('test apiServerId', () async {
      // TODO
    });

    // Returns the peers connected to the host via the transport layer.
    //
    // As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```
    //
    //Future<Map<String, List<String>>> apiServerPeers() async
    test('test apiServerPeers', () async {
      // TODO
    });

    // Returns the build version running on the server.
    //
    // See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.
    //
    //Future<PublicapiVersionResponse> apiServerVersion(PublicapiVersionRequest body) async
    test('test apiServerVersion', () async {
      // TODO
    });

  });
}
