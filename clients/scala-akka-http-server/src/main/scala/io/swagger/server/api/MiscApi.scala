package io.swagger.server.api

import akka.http.scaladsl.server.Directives._
import akka.http.scaladsl.server.Route
import akka.http.scaladsl.unmarshalling.FromRequestUnmarshaller
import akka.http.scaladsl.marshalling.ToEntityMarshaller
import io.swagger.server.AkkaHttpHelper._
import io.swagger.server.model.Publicapi.versionRequest
import io.swagger.server.model.publicapi.versionResponse

class MiscApi(
    miscService: MiscApiService,
    miscMarshaller: MiscApiMarshaller
) {
  import miscMarshaller._

  lazy val route: Route =
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  miscService.apiServer/id()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      get {
        parameters() { () =>
          
            formFields() { () =>
              
                
                  miscService.apiServer/peers()
               
             
            }
         
        }
      }
    } ~
    path() { () => 
      post {
        parameters() { () =>
          
            formFields() { () =>
              
                entity(as[Publicapi.versionRequest]){ body =>
                  miscService.apiServer/version(body = body)
                }
             
            }
         
        }
      }
    }
}

trait MiscApiService {

  def apiServer/id200(responseString: String): Route =
    complete((200, responseString))
  def apiServer/id500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def apiServer/id()
      (implicit ): Route

  def apiServer/peers200(responseMapmap: Map[String, List[String]]): Route =
    complete((200, responseMapmap))
  def apiServer/peers500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: Map[String, List[String]]
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def apiServer/peers()
      (implicit ): Route

  def apiServer/version200(responsepublicapi.versionResponse: publicapi.versionResponse)(implicit toEntityMarshallerpublicapi.versionResponse: ToEntityMarshaller[publicapi.versionResponse]): Route =
    complete((200, responsepublicapi.versionResponse))
  def apiServer/version400(responseString: String): Route =
    complete((400, responseString))
  def apiServer/version500(responseString: String): Route =
    complete((500, responseString))
  /**
   * Code: 200, Message: OK, DataType: publicapi.versionResponse
   * Code: 400, Message: Bad Request, DataType: String
   * Code: 500, Message: Internal Server Error, DataType: String
   */
  def apiServer/version(body: Publicapi.versionRequest)
      (implicit toEntityMarshallerpublicapi.versionResponse: ToEntityMarshaller[publicapi.versionResponse]): Route

}

trait MiscApiMarshaller {
  implicit def fromRequestUnmarshallerPublicapi.versionRequest: FromRequestUnmarshaller[Publicapi.versionRequest]


  implicit def toEntityMarshallerpublicapi.versionResponse: ToEntityMarshaller[publicapi.versionResponse]

}

