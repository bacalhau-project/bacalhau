package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * RequesternodeActiveJob
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class RequesternodeActiveJob   {
  @JsonProperty("BiddingNodesCount")
  private Integer biddingNodesCount = null;

  @JsonProperty("CompletedNodesCount")
  private Integer completedNodesCount = null;

  @JsonProperty("ShardID")
  private String shardID = null;

  @JsonProperty("State")
  private String state = null;

  public RequesternodeActiveJob biddingNodesCount(Integer biddingNodesCount) {
    this.biddingNodesCount = biddingNodesCount;
    return this;
  }

  /**
   * Get biddingNodesCount
   * @return biddingNodesCount
   **/
  @Schema(description = "")
  
    public Integer getBiddingNodesCount() {
    return biddingNodesCount;
  }

  public void setBiddingNodesCount(Integer biddingNodesCount) {
    this.biddingNodesCount = biddingNodesCount;
  }

  public RequesternodeActiveJob completedNodesCount(Integer completedNodesCount) {
    this.completedNodesCount = completedNodesCount;
    return this;
  }

  /**
   * Get completedNodesCount
   * @return completedNodesCount
   **/
  @Schema(description = "")
  
    public Integer getCompletedNodesCount() {
    return completedNodesCount;
  }

  public void setCompletedNodesCount(Integer completedNodesCount) {
    this.completedNodesCount = completedNodesCount;
  }

  public RequesternodeActiveJob shardID(String shardID) {
    this.shardID = shardID;
    return this;
  }

  /**
   * Get shardID
   * @return shardID
   **/
  @Schema(description = "")
  
    public String getShardID() {
    return shardID;
  }

  public void setShardID(String shardID) {
    this.shardID = shardID;
  }

  public RequesternodeActiveJob state(String state) {
    this.state = state;
    return this;
  }

  /**
   * Get state
   * @return state
   **/
  @Schema(description = "")
  
    public String getState() {
    return state;
  }

  public void setState(String state) {
    this.state = state;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    RequesternodeActiveJob requesternodeActiveJob = (RequesternodeActiveJob) o;
    return Objects.equals(this.biddingNodesCount, requesternodeActiveJob.biddingNodesCount) &&
        Objects.equals(this.completedNodesCount, requesternodeActiveJob.completedNodesCount) &&
        Objects.equals(this.shardID, requesternodeActiveJob.shardID) &&
        Objects.equals(this.state, requesternodeActiveJob.state);
  }

  @Override
  public int hashCode() {
    return Objects.hash(biddingNodesCount, completedNodesCount, shardID, state);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class RequesternodeActiveJob {\n");
    
    sb.append("    biddingNodesCount: ").append(toIndentedString(biddingNodesCount)).append("\n");
    sb.append("    completedNodesCount: ").append(toIndentedString(completedNodesCount)).append("\n");
    sb.append("    shardID: ").append(toIndentedString(shardID)).append("\n");
    sb.append("    state: ").append(toIndentedString(state)).append("\n");
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
