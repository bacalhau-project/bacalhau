version := "1.0.0"
name := "swagger-scala-akka-http-server"
organization := "io.swagger"
scalaVersion := "2.12.6"

libraryDependencies ++= Seq(
  "com.typesafe.akka" %% "akka-http" % "10.1.5",
  "com.typesafe.akka" %% "akka-stream" % "2.5.16",
)
