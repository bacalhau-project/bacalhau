package io.swagger.server

import akka.http.scaladsl.Http
import akka.http.scaladsl.server.Route
import io.swagger.server.api.HealthApi
import io.swagger.server.api.JobApi
import io.swagger.server.api.MiscApi
import akka.http.scaladsl.server.Directives._
import akka.actor.ActorSystem
import akka.stream.ActorMaterializer

class Controller(health: HealthApi, job: JobApi, misc: MiscApi)(implicit system: ActorSystem, materializer: ActorMaterializer) {

    lazy val routes: Route = health.route ~ job.route ~ misc.route 

    Http().bindAndHandle(routes, "0.0.0.0", 9000)
}