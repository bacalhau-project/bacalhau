package io.swagger.server.api

import akka.http.scaladsl.server.Directives._
import akka.http.scaladsl.server.Route
import akka.http.scaladsl.unmarshalling.FromRequestUnmarshaller
import akka.http.scaladsl.marshalling.ToEntityMarshaller
import io.swagger.server.AkkaHttpHelper._
import io.swagger.server.model.Publicapi.eventsRequest
import io.swagger.server.model.Publicapi.listRequest
import io.swagger.server.model.Publicapi.localEventsRequest
import io.swagger.server.model.Publicapi.stateRequest
import io.swagger.server.model.Publicapi.submitRequest
import io.swagger.server.model.publicapi.eventsResponse
import io.swagger.server.model.publicapi.listResponse
import io.swagger.server.model.publicapi.localEventsResponse
import io.swagger.server.model.publicapi.resultsResponse
import io.swagger.server.model.publicapi.stateResponse
import io.swagger.server.model.publicapi.submitResponse

class JobApi(
    jobService: JobApiService,
    jobMarshaller: JobApiMarshaller
) {
  import jobMarshaller._

  lazy val route: Route =
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.submitRequest]){ body =>
                  jobService.pkg/apiServer.submit(body = body)
                }
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.listRequest]){ body =>
                  jobService.pkg/publicapi.list(body = body)
                }
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.eventsRequest]){ body =>
                  jobService.pkg/publicapi/events(body = body)
                }
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.localEventsRequest]){ body =>
                  jobService.pkg/publicapi/localEvents(body = body)
                }
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.stateRequest]){ body =>
                  jobService.pkg/publicapi/results(body = body)
                }
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.stateRequest]){ body =>
                  jobService.pkg/publicapi/states(body = body)
                }
             
            }
         
        }
      }
    }
}

trait JobApiService {

  def pkg/apiServer.submit200(responsepublicapi.submitResponse: publicapi.submitResponse)(implicit toEntityMarshallerpublicapi.submitResponse: ToEntityMarshaller[publicapi.submitResponse]): Route =
    complete((200, responsepublicapi.submitResponse))
  def pkg/apiServer.submit400(responseString: String): Route =
    complete((400, responseString))
  def pkg/apiServer.submit500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.submitResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/apiServer.submit(body: Publicapi.submitRequest)
      (implicit toEntityMarshallerpublicapi.submitResponse: ToEntityMarshaller[publicapi.submitResponse]): Route

  def pkg/publicapi.list200(responsepublicapi.listResponse: publicapi.listResponse)(implicit toEntityMarshallerpublicapi.listResponse: ToEntityMarshaller[publicapi.listResponse]): Route =
    complete((200, responsepublicapi.listResponse))
  def pkg/publicapi.list400(responseString: String): Route =
    complete((400, responseString))
  def pkg/publicapi.list500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.listResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/publicapi.list(body: Publicapi.listRequest)
      (implicit toEntityMarshallerpublicapi.listResponse: ToEntityMarshaller[publicapi.listResponse]): Route

  def pkg/publicapi/events200(responsepublicapi.eventsResponse: publicapi.eventsResponse)(implicit toEntityMarshallerpublicapi.eventsResponse: ToEntityMarshaller[publicapi.eventsResponse]): Route =
    complete((200, responsepublicapi.eventsResponse))
  def pkg/publicapi/events400(responseString: String): Route =
    complete((400, responseString))
  def pkg/publicapi/events500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.eventsResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/publicapi/events(body: Publicapi.eventsRequest)
      (implicit toEntityMarshallerpublicapi.eventsResponse: ToEntityMarshaller[publicapi.eventsResponse]): Route

  def pkg/publicapi/localEvents200(responsepublicapi.localEventsResponse: publicapi.localEventsResponse)(implicit toEntityMarshallerpublicapi.localEventsResponse: ToEntityMarshaller[publicapi.localEventsResponse]): Route =
    complete((200, responsepublicapi.localEventsResponse))
  def pkg/publicapi/localEvents400(responseString: String): Route =
    complete((400, responseString))
  def pkg/publicapi/localEvents500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.localEventsResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/publicapi/localEvents(body: Publicapi.localEventsRequest)
      (implicit toEntityMarshallerpublicapi.localEventsResponse: ToEntityMarshaller[publicapi.localEventsResponse]): Route

  def pkg/publicapi/results200(responsepublicapi.resultsResponse: publicapi.resultsResponse)(implicit toEntityMarshallerpublicapi.resultsResponse: ToEntityMarshaller[publicapi.resultsResponse]): Route =
    complete((200, responsepublicapi.resultsResponse))
  def pkg/publicapi/results400(responseString: String): Route =
    complete((400, responseString))
  def pkg/publicapi/results500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.resultsResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/publicapi/results(body: Publicapi.stateRequest)
      (implicit toEntityMarshallerpublicapi.resultsResponse: ToEntityMarshaller[publicapi.resultsResponse]): Route

  def pkg/publicapi/states200(responsepublicapi.stateResponse: publicapi.stateResponse)(implicit toEntityMarshallerpublicapi.stateResponse: ToEntityMarshaller[publicapi.stateResponse]): Route =
    complete((200, responsepublicapi.stateResponse))
  def pkg/publicapi/states400(responseString: String): Route =
    complete((400, responseString))
  def pkg/publicapi/states500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.stateResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def pkg/publicapi/states(body: Publicapi.stateRequest)
      (implicit toEntityMarshallerpublicapi.stateResponse: ToEntityMarshaller[publicapi.stateResponse]): Route

}

trait JobApiMarshaller {
  implicit def fromRequestUnmarshallerPublicapi.stateRequest: FromRequestUnmarshaller[Publicapi.stateRequest]

  implicit def fromRequestUnmarshallerPublicapi.eventsRequest: FromRequestUnmarshaller[Publicapi.eventsRequest]

  implicit def fromRequestUnmarshallerPublicapi.localEventsRequest: FromRequestUnmarshaller[Publicapi.localEventsRequest]

  implicit def fromRequestUnmarshallerPublicapi.listRequest: FromRequestUnmarshaller[Publicapi.listRequest]

  implicit def fromRequestUnmarshallerPublicapi.submitRequest: FromRequestUnmarshaller[Publicapi.submitRequest]


  implicit def toEntityMarshallerpublicapi.submitResponse: ToEntityMarshaller[publicapi.submitResponse]

  implicit def toEntityMarshallerpublicapi.listResponse: ToEntityMarshaller[publicapi.listResponse]

  implicit def toEntityMarshallerpublicapi.eventsResponse: ToEntityMarshaller[publicapi.eventsResponse]

  implicit def toEntityMarshallerpublicapi.localEventsResponse: ToEntityMarshaller[publicapi.localEventsResponse]

  implicit def toEntityMarshallerpublicapi.resultsResponse: ToEntityMarshaller[publicapi.resultsResponse]

  implicit def toEntityMarshallerpublicapi.stateResponse: ToEntityMarshaller[publicapi.stateResponse]

}

