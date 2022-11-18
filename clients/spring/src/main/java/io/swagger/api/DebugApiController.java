package io.swagger.api;

import io.swagger.model.PublicapiDebugResponse;
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
public class DebugApiController implements DebugApi {

    private static final Logger log = LoggerFactory.getLogger(DebugApiController.class);

    private final ObjectMapper objectMapper;

    private final HttpServletRequest request;

    @org.springframework.beans.factory.annotation.Autowired
    public DebugApiController(ObjectMapper objectMapper, HttpServletRequest request) {
        this.objectMapper = objectMapper;
        this.request = request;
    }

    public ResponseEntity<PublicapiDebugResponse> apiServerdebug() {
        String accept = request.getHeader("Accept");
        if (accept != null && accept.contains("application/json")) {
            try {
                return new ResponseEntity<PublicapiDebugResponse>(objectMapper.readValue("{\n  \"ComputeJobs\" : [ {\n    \"ShardID\" : \"ShardID\",\n    \"State\" : \"State\"\n  }, {\n    \"ShardID\" : \"ShardID\",\n    \"State\" : \"State\"\n  } ],\n  \"AvailableComputeCapacity\" : {\n    \"Memory\" : 27487790694,\n    \"CPU\" : 9.600000000000001,\n    \"Disk\" : 212663867801,\n    \"GPU\" : 1\n  },\n  \"RequesterJobs\" : [ {\n    \"ShardID\" : \"ShardID\",\n    \"State\" : \"State\",\n    \"CompletedNodesCount\" : 6,\n    \"BiddingNodesCount\" : 0\n  }, {\n    \"ShardID\" : \"ShardID\",\n    \"State\" : \"State\",\n    \"CompletedNodesCount\" : 6,\n    \"BiddingNodesCount\" : 0\n  } ]\n}", PublicapiDebugResponse.class), HttpStatus.NOT_IMPLEMENTED);
            } catch (IOException e) {
                log.error("Couldn't serialize response for content type application/json", e);
                return new ResponseEntity<PublicapiDebugResponse>(HttpStatus.INTERNAL_SERVER_ERROR);
            }
        }

        return new ResponseEntity<PublicapiDebugResponse>(HttpStatus.NOT_IMPLEMENTED);
    }

}
