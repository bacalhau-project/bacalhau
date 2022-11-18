package io.swagger.api;

import io.swagger.model.PublicapiStateRequest;
import io.swagger.model.PublicapiStateResponse;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.enums.ParameterIn;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.media.ArraySchema;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.CookieValue;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import javax.validation.constraints.*;
import javax.validation.Valid;
import javax.servlet.http.HttpServletRequest;
import java.io.IOException;
import java.util.List;
import java.util.Map;

@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")
@RestController
public class StatesApiController implements StatesApi {

    private static final Logger log = LoggerFactory.getLogger(StatesApiController.class);

    private final ObjectMapper objectMapper;

    private final HttpServletRequest request;

    @org.springframework.beans.factory.annotation.Autowired
    public StatesApiController(ObjectMapper objectMapper, HttpServletRequest request) {
        this.objectMapper = objectMapper;
        this.request = request;
    }

    public ResponseEntity<PublicapiStateResponse> pkgpublicapistates(@Parameter(in = ParameterIn.DEFAULT, description = "", required=true, schema=@Schema()) @Valid @RequestBody PublicapiStateRequest body) {
        String accept = request.getHeader("Accept");
        if (accept != null && accept.contains("application/json")) {
            try {
                return new ResponseEntity<PublicapiStateResponse>(objectMapper.readValue("{\n  \"state\" : {\n    \"Nodes\" : {\n      \"key\" : {\n        \"Shards\" : {\n          \"key\" : {\n            \"Status\" : \"Status\",\n            \"RunOutput\" : {\n              \"stderrtruncated\" : true,\n              \"stdout\" : \"stdout\",\n              \"exitCode\" : 7,\n              \"runnerError\" : \"runnerError\",\n              \"stdouttruncated\" : true,\n              \"stderr\" : \"stderr\"\n            },\n            \"VerificationProposal\" : [ 1, 1 ],\n            \"VerificationResult\" : {\n              \"Complete\" : true,\n              \"Result\" : true\n            },\n            \"ShardIndex\" : 0,\n            \"State\" : 6,\n            \"NodeId\" : \"NodeId\",\n            \"PublishedResults\" : {\n              \"path\" : \"path\",\n              \"Metadata\" : {\n                \"key\" : \"Metadata\"\n              },\n              \"URL\" : \"URL\",\n              \"CID\" : \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\",\n              \"Name\" : \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n              \"StorageSource\" : 2\n            }\n          }\n        }\n      }\n    }\n  }\n}", PublicapiStateResponse.class), HttpStatus.NOT_IMPLEMENTED);
            } catch (IOException e) {
                log.error("Couldn't serialize response for content type application/json", e);
                return new ResponseEntity<PublicapiStateResponse>(HttpStatus.INTERNAL_SERVER_ERROR);
            }
        }

        return new ResponseEntity<PublicapiStateResponse>(HttpStatus.NOT_IMPLEMENTED);
    }

}
