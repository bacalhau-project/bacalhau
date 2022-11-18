package io.swagger.server.api

import akka.http.scaladsl.server.Directives._
import akka.http.scaladsl.server.Route
import akka.http.scaladsl.unmarshalling.FromRequestUnmarshaller
import akka.http.scaladsl.marshalling.ToEntityMarshaller
import io.swagger.server.AkkaHttpHelper._
import io.swagger.server.model.publicapi.debugResponse
import io.swagger.server.model.types.HealthInfo

class HealthApi(
    healthService: HealthApiService,
    healthMarshaller: HealthApiMarshaller
) {
  import healthMarshaller._

  lazy val route: Route =
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/debug()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/healthz()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/livez()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/logz()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/readyz()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  healthService.apiServer/varz()
               
             
            }
         
        }
      }
    }
}

trait HealthApiService {

  def apiServer/debug200(responsepublicapi.debugResponse: publicapi.debugResponse)(implicit toEntityMarshallerpublicapi.debugResponse: ToEntityMarshaller[publicapi.debugResponse]): Route =
    complete((200, responsepublicapi.debugResponse))
  def apiServer/debug500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.debugResponse
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def apiServer/debug()
      (implicit toEntityMarshallerpublicapi.debugResponse: ToEntityMarshaller[publicapi.debugResponse]): Route

  def apiServer/healthz200(responsetypes.HealthInfo: types.HealthInfo)(implicit toEntityMarshallertypes.HealthInfo: ToEntityMarshaller[types.HealthInfo]): Route =
    complete((200, responsetypes.HealthInfo))
  /**
   * Code: 200, Message: OK, DataType: types.HealthInfo
   */
  def apiServer/healthz()
      (implicit toEntityMarshallertypes.HealthInfo: ToEntityMarshaller[types.HealthInfo]): Route

  def apiServer/livez200(responseString: String): Route =
    complete((200, responseString))
  /**
   * Code: 200, Message: TODO, DataType: String
   */
  def apiServer/livez()
      (implicit ): Route

  def apiServer/logz200(responseString: String): Route =
    complete((200, responseString))
  /**
   * Code: 200, Message: TODO, DataType: String
   */
  def apiServer/logz()
      (implicit ): Route

  def apiServer/readyz200(responseString: String): Route =
    complete((200, responseString))
  /**
   * Code: 200, Message: OK, DataType: String
   */
  def apiServer/readyz()
      (implicit ): Route

  def apiServer/varz200(responseIntarray: List[Int]): Route =
    complete((200, responseIntarray))
  /**
   * Code: 200, Message: OK, DataType: List[Int]
   */
  def apiServer/varz()
      (implicit ): Route

}

trait HealthApiMarshaller {

  implicit def toEntityMarshallerpublicapi.debugResponse: ToEntityMarshaller[publicapi.debugResponse]

  implicit def toEntityMarshallertypes.HealthInfo: ToEntityMarshaller[types.HealthInfo]

}

