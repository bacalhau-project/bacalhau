package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJob;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobCreatePayload
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobCreatePayload   {
  @JsonProperty("ClientID")
  private String clientID = null;

  @JsonProperty("Context")
  private String context = null;

  @JsonProperty("Job")
  private ModelJob job = null;

  public ModelJobCreatePayload clientID(String clientID) {
    this.clientID = clientID;
    return this;
  }

  /**
   * the id of the client that is submitting the job
   * @return clientID
   **/
  @Schema(required = true, description = "the id of the client that is submitting the job")
      @NotNull

    public String getClientID() {
    return clientID;
  }

  public void setClientID(String clientID) {
    this.clientID = clientID;
  }

  public ModelJobCreatePayload context(String context) {
    this.context = context;
    return this;
  }

  /**
   * Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don't flood the transport layer with it (potentially very large).
   * @return context
   **/
  @Schema(description = "Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don't flood the transport layer with it (potentially very large).")
  
    public String getContext() {
    return context;
  }

  public void setContext(String context) {
    this.context = context;
  }

  public ModelJobCreatePayload job(ModelJob job) {
    this.job = job;
    return this;
  }

  /**
   * Get job
   * @return job
   **/
  @Schema(required = true, description = "")
      @NotNull

    @Valid
    public ModelJob getJob() {
    return job;
  }

  public void setJob(ModelJob job) {
    this.job = job;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobCreatePayload modelJobCreatePayload = (ModelJobCreatePayload) o;
    return Objects.equals(this.clientID, modelJobCreatePayload.clientID) &&
        Objects.equals(this.context, modelJobCreatePayload.context) &&
        Objects.equals(this.job, modelJobCreatePayload.job);
  }

  @Override
  public int hashCode() {
    return Objects.hash(clientID, context, job);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobCreatePayload {\n");
    
    sb.append("    clientID: ").append(toIndentedString(clientID)).append("\n");
    sb.append("    context: ").append(toIndentedString(context)).append("\n");
    sb.append("    job: ").append(toIndentedString(job)).append("\n");
    sb.append("}");
    return sb.toString();
  }

  /**
   * Convert the given object to string with each line indented by 4 spaces
   * (except the first line).
   */
  private String toIndentedString(java.lang.Object o) {
    if (o == null) {
      return "null";
    }
    return o.toString().replace("\n", "\n    ");
  }
}
