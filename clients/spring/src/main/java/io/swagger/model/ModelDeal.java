package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelDeal
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelDeal   {
  @JsonProperty("Concurrency")
  private Integer concurrency = null;

  @JsonProperty("Confidence")
  private Integer confidence = null;

  @JsonProperty("MinBids")
  private Integer minBids = null;

  public ModelDeal concurrency(Integer concurrency) {
    this.concurrency = concurrency;
    return this;
  }

  /**
   * The maximum number of concurrent compute node bids that will be accepted by the requester node on behalf of the client.
   * @return concurrency
   **/
  @Schema(description = "The maximum number of concurrent compute node bids that will be accepted by the requester node on behalf of the client.")
  
    public Integer getConcurrency() {
    return concurrency;
  }

  public void setConcurrency(Integer concurrency) {
    this.concurrency = concurrency;
  }

  public ModelDeal confidence(Integer confidence) {
    this.confidence = confidence;
    return this;
  }

  /**
   * The number of nodes that must agree on a verification result this is used by the different verifiers - for example the deterministic verifier requires the winning group size to be at least this size
   * @return confidence
   **/
  @Schema(description = "The number of nodes that must agree on a verification result this is used by the different verifiers - for example the deterministic verifier requires the winning group size to be at least this size")
  
    public Integer getConfidence() {
    return confidence;
  }

  public void setConfidence(Integer confidence) {
    this.confidence = confidence;
  }

  public ModelDeal minBids(Integer minBids) {
    this.minBids = minBids;
    return this;
  }

  /**
   * The minimum number of bids that must be received before the Requester node will randomly accept concurrency-many of them. This allows the Requester node to get some level of guarantee that the execution of the jobs will be spread evenly across the network (assuming that this value is some large proportion of the size of the network).
   * @return minBids
   **/
  @Schema(description = "The minimum number of bids that must be received before the Requester node will randomly accept concurrency-many of them. This allows the Requester node to get some level of guarantee that the execution of the jobs will be spread evenly across the network (assuming that this value is some large proportion of the size of the network).")
  
    public Integer getMinBids() {
    return minBids;
  }

  public void setMinBids(Integer minBids) {
    this.minBids = minBids;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelDeal modelDeal = (ModelDeal) o;
    return Objects.equals(this.concurrency, modelDeal.concurrency) &&
        Objects.equals(this.confidence, modelDeal.confidence) &&
        Objects.equals(this.minBids, modelDeal.minBids);
  }

  @Override
  public int hashCode() {
    return Objects.hash(concurrency, confidence, minBids);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelDeal {\n");
    
    sb.append("    concurrency: ").append(toIndentedString(concurrency)).append("\n");
    sb.append("    confidence: ").append(toIndentedString(confidence)).append("\n");
    sb.append("    minBids: ").append(toIndentedString(minBids)).append("\n");
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
