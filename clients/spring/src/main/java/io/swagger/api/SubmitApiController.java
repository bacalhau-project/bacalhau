package io.swagger.api;

import io.swagger.model.PublicapiSubmitRequest;
import io.swagger.model.PublicapiSubmitResponse;
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
public class SubmitApiController implements SubmitApi {

    private static final Logger log = LoggerFactory.getLogger(SubmitApiController.class);

    private final ObjectMapper objectMapper;

    private final HttpServletRequest request;

    @org.springframework.beans.factory.annotation.Autowired
    public SubmitApiController(ObjectMapper objectMapper, HttpServletRequest request) {
        this.objectMapper = objectMapper;
        this.request = request;
    }

    public ResponseEntity<PublicapiSubmitResponse> pkgapiServerSubmit(@Parameter(in = ParameterIn.DEFAULT, description = "", required=true, schema=@Schema()) @Valid @RequestBody PublicapiSubmitRequest body) {
        String accept = request.getHeader("Accept");
        if (accept != null && accept.contains("application/json")) {
            try {
                return new ResponseEntity<PublicapiSubmitResponse>(objectMapper.readValue("{\n  \"job\" : {\n    \"RequesterNodeID\" : \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",\n    \"ExecutionPlan\" : {\n      \"ShardsTotal\" : 5\n    },\n    \"LocalJobEvents\" : [ {\n      \"TargetNodeID\" : \"TargetNodeID\",\n      \"ShardIndex\" : 5,\n      \"EventName\" : 5,\n      \"JobID\" : \"JobID\"\n    }, {\n      \"TargetNodeID\" : \"TargetNodeID\",\n      \"ShardIndex\" : 5,\n      \"EventName\" : 5,\n      \"JobID\" : \"JobID\"\n    } ],\n    \"APIVersion\" : \"V1beta1\",\n    \"CreatedAt\" : \"2022-11-17T13:29:01.871140291Z\",\n    \"ClientID\" : \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",\n    \"JobState\" : {\n      \"Nodes\" : {\n        \"key\" : {\n          \"Shards\" : {\n            \"key\" : {\n              \"Status\" : \"Status\",\n              \"RunOutput\" : {\n                \"stderrtruncated\" : true,\n                \"stdout\" : \"stdout\",\n                \"exitCode\" : 7,\n                \"runnerError\" : \"runnerError\",\n                \"stdouttruncated\" : true,\n                \"stderr\" : \"stderr\"\n              },\n              \"VerificationProposal\" : [ 1, 1 ],\n              \"VerificationResult\" : {\n                \"Complete\" : true,\n                \"Result\" : true\n              },\n              \"ShardIndex\" : 0,\n              \"State\" : 6,\n              \"NodeId\" : \"NodeId\",\n              \"PublishedResults\" : {\n                \"path\" : \"path\",\n                \"Metadata\" : {\n                  \"key\" : \"Metadata\"\n                },\n                \"URL\" : \"URL\",\n                \"CID\" : \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\",\n                \"Name\" : \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n                \"StorageSource\" : 2\n              }\n            }\n          }\n        }\n      }\n    },\n    \"RequesterPublicKey\" : [ 2, 2 ],\n    \"ID\" : \"92d5d4ee-3765-4f78-8353-623f5f26df08\",\n    \"Deal\" : {\n      \"MinBids\" : 1,\n      \"Concurrency\" : 0,\n      \"Confidence\" : 6\n    },\n    \"JobEvents\" : [ {\n      \"Status\" : \"Got results proposal of length: 0\",\n      \"JobExecutionPlan\" : {\n        \"ShardsTotal\" : 5\n      },\n      \"SourceNodeID\" : \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",\n      \"ShardIndex\" : 3,\n      \"EventName\" : 5,\n      \"Deal\" : {\n        \"MinBids\" : 1,\n        \"Concurrency\" : 0,\n        \"Confidence\" : 6\n      },\n      \"PublishedResult\" : {\n        \"path\" : \"path\",\n        \"Metadata\" : {\n          \"key\" : \"Metadata\"\n        },\n        \"URL\" : \"URL\",\n        \"CID\" : \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\",\n        \"Name\" : \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n        \"StorageSource\" : 2\n      },\n      \"TargetNodeID\" : \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n      \"RunOutput\" : {\n        \"stderrtruncated\" : true,\n        \"stdout\" : \"stdout\",\n        \"exitCode\" : 7,\n        \"runnerError\" : \"runnerError\",\n        \"stdouttruncated\" : true,\n        \"stderr\" : \"stderr\"\n      },\n      \"VerificationProposal\" : [ 1, 1 ],\n      \"VerificationResult\" : {\n        \"Complete\" : true,\n        \"Result\" : true\n      },\n      \"APIVersion\" : \"V1beta1\",\n      \"SenderPublicKey\" : [ 9, 9 ],\n      \"EventTime\" : \"2022-11-17T13:32:55.756658941Z\",\n      \"ClientID\" : \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",\n      \"Spec\" : {\n        \"outputs\" : [ null, null ],\n        \"Sharding\" : {\n          \"BatchSize\" : 7,\n          \"GlobPattern\" : \"GlobPattern\",\n          \"GlobPatternBasePath\" : \"GlobPatternBasePath\"\n        },\n        \"Timeout\" : 1.2315135367772556,\n        \"inputs\" : [ null, null ],\n        \"DoNotTrack\" : true,\n        \"Publisher\" : 4,\n        \"Verifier\" : 1,\n        \"Contexts\" : [ null, null ],\n        \"Wasm\" : {\n          \"EnvironmentVariables\" : {\n            \"key\" : \"EnvironmentVariables\"\n          },\n          \"Parameters\" : [ \"Parameters\", \"Parameters\" ],\n          \"ImportModules\" : [ null, null ],\n          \"EntryPoint\" : \"EntryPoint\"\n        },\n        \"Annotations\" : [ \"Annotations\", \"Annotations\" ],\n        \"Docker\" : {\n          \"WorkingDirectory\" : \"WorkingDirectory\",\n          \"EnvironmentVariables\" : [ \"EnvironmentVariables\", \"EnvironmentVariables\" ],\n          \"Entrypoint\" : [ \"Entrypoint\", \"Entrypoint\" ],\n          \"Image\" : \"Image\"\n        },\n        \"Language\" : {\n          \"RequirementsPath\" : \"RequirementsPath\",\n          \"Language\" : \"Language\",\n          \"Command\" : \"Command\",\n          \"DeterministicExecution\" : true,\n          \"LanguageVersion\" : \"LanguageVersion\",\n          \"ProgramPath\" : \"ProgramPath\"\n        },\n        \"Resources\" : {\n          \"Memory\" : \"Memory\",\n          \"CPU\" : \"CPU\",\n          \"Disk\" : \"Disk\",\n          \"GPU\" : \"GPU\"\n        },\n        \"Engine\" : 2\n      },\n      \"JobID\" : \"9304c616-291f-41ad-b862-54e133c0149e\"\n    }, {\n      \"Status\" : \"Got results proposal of length: 0\",\n      \"JobExecutionPlan\" : {\n        \"ShardsTotal\" : 5\n      },\n      \"SourceNodeID\" : \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",\n      \"ShardIndex\" : 3,\n      \"EventName\" : 5,\n      \"Deal\" : {\n        \"MinBids\" : 1,\n        \"Concurrency\" : 0,\n        \"Confidence\" : 6\n      },\n      \"PublishedResult\" : {\n        \"path\" : \"path\",\n        \"Metadata\" : {\n          \"key\" : \"Metadata\"\n        },\n        \"URL\" : \"URL\",\n        \"CID\" : \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\",\n        \"Name\" : \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n        \"StorageSource\" : 2\n      },\n      \"TargetNodeID\" : \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",\n      \"RunOutput\" : {\n        \"stderrtruncated\" : true,\n        \"stdout\" : \"stdout\",\n        \"exitCode\" : 7,\n        \"runnerError\" : \"runnerError\",\n        \"stdouttruncated\" : true,\n        \"stderr\" : \"stderr\"\n      },\n      \"VerificationProposal\" : [ 1, 1 ],\n      \"VerificationResult\" : {\n        \"Complete\" : true,\n        \"Result\" : true\n      },\n      \"APIVersion\" : \"V1beta1\",\n      \"SenderPublicKey\" : [ 9, 9 ],\n      \"EventTime\" : \"2022-11-17T13:32:55.756658941Z\",\n      \"ClientID\" : \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",\n      \"Spec\" : {\n        \"outputs\" : [ null, null ],\n        \"Sharding\" : {\n          \"BatchSize\" : 7,\n          \"GlobPattern\" : \"GlobPattern\",\n          \"GlobPatternBasePath\" : \"GlobPatternBasePath\"\n        },\n        \"Timeout\" : 1.2315135367772556,\n        \"inputs\" : [ null, null ],\n        \"DoNotTrack\" : true,\n        \"Publisher\" : 4,\n        \"Verifier\" : 1,\n        \"Contexts\" : [ null, null ],\n        \"Wasm\" : {\n          \"EnvironmentVariables\" : {\n            \"key\" : \"EnvironmentVariables\"\n          },\n          \"Parameters\" : [ \"Parameters\", \"Parameters\" ],\n          \"ImportModules\" : [ null, null ],\n          \"EntryPoint\" : \"EntryPoint\"\n        },\n        \"Annotations\" : [ \"Annotations\", \"Annotations\" ],\n        \"Docker\" : {\n          \"WorkingDirectory\" : \"WorkingDirectory\",\n          \"EnvironmentVariables\" : [ \"EnvironmentVariables\", \"EnvironmentVariables\" ],\n          \"Entrypoint\" : [ \"Entrypoint\", \"Entrypoint\" ],\n          \"Image\" : \"Image\"\n        },\n        \"Language\" : {\n          \"RequirementsPath\" : \"RequirementsPath\",\n          \"Language\" : \"Language\",\n          \"Command\" : \"Command\",\n          \"DeterministicExecution\" : true,\n          \"LanguageVersion\" : \"LanguageVersion\",\n          \"ProgramPath\" : \"ProgramPath\"\n        },\n        \"Resources\" : {\n          \"Memory\" : \"Memory\",\n          \"CPU\" : \"CPU\",\n          \"Disk\" : \"Disk\",\n          \"GPU\" : \"GPU\"\n        },\n        \"Engine\" : 2\n      },\n      \"JobID\" : \"9304c616-291f-41ad-b862-54e133c0149e\"\n    } ],\n    \"Spec\" : {\n      \"outputs\" : [ null, null ],\n      \"Sharding\" : {\n        \"BatchSize\" : 7,\n        \"GlobPattern\" : \"GlobPattern\",\n        \"GlobPatternBasePath\" : \"GlobPatternBasePath\"\n      },\n      \"Timeout\" : 1.2315135367772556,\n      \"inputs\" : [ null, null ],\n      \"DoNotTrack\" : true,\n      \"Publisher\" : 4,\n      \"Verifier\" : 1,\n      \"Contexts\" : [ null, null ],\n      \"Wasm\" : {\n        \"EnvironmentVariables\" : {\n          \"key\" : \"EnvironmentVariables\"\n        },\n        \"Parameters\" : [ \"Parameters\", \"Parameters\" ],\n        \"ImportModules\" : [ null, null ],\n        \"EntryPoint\" : \"EntryPoint\"\n      },\n      \"Annotations\" : [ \"Annotations\", \"Annotations\" ],\n      \"Docker\" : {\n        \"WorkingDirectory\" : \"WorkingDirectory\",\n        \"EnvironmentVariables\" : [ \"EnvironmentVariables\", \"EnvironmentVariables\" ],\n        \"Entrypoint\" : [ \"Entrypoint\", \"Entrypoint\" ],\n        \"Image\" : \"Image\"\n      },\n      \"Language\" : {\n        \"RequirementsPath\" : \"RequirementsPath\",\n        \"Language\" : \"Language\",\n        \"Command\" : \"Command\",\n        \"DeterministicExecution\" : true,\n        \"LanguageVersion\" : \"LanguageVersion\",\n        \"ProgramPath\" : \"ProgramPath\"\n      },\n      \"Resources\" : {\n        \"Memory\" : \"Memory\",\n        \"CPU\" : \"CPU\",\n        \"Disk\" : \"Disk\",\n        \"GPU\" : \"GPU\"\n      },\n      \"Engine\" : 2\n    }\n  }\n}", PublicapiSubmitResponse.class), HttpStatus.NOT_IMPLEMENTED);
            } catch (IOException e) {
                log.error("Couldn't serialize response for content type application/json", e);
                return new ResponseEntity<PublicapiSubmitResponse>(HttpStatus.INTERNAL_SERVER_ERROR);
            }
        }

        return new ResponseEntity<PublicapiSubmitResponse>(HttpStatus.NOT_IMPLEMENTED);
    }

}
