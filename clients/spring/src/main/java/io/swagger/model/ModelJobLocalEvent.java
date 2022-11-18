package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobLocalEvent
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobLocalEvent   {
  @JsonProperty("EventName")
  private Integer eventName = null;

  @JsonProperty("JobID")
  private String jobID = null;

  @JsonProperty("ShardIndex")
  private Integer shardIndex = null;

  @JsonProperty("TargetNodeID")
  private String targetNodeID = null;

  public ModelJobLocalEvent eventName(Integer eventName) {
    this.eventName = eventName;
    return this;
  }

  /**
   * Get eventName
   * @return eventName
   **/
  @Schema(description = "")
  
    public Integer getEventName() {
    return eventName;
  }

  public void setEventName(Integer eventName) {
    this.eventName = eventName;
  }

  public ModelJobLocalEvent jobID(String jobID) {
    this.jobID = jobID;
    return this;
  }

  /**
   * Get jobID
   * @return jobID
   **/
  @Schema(description = "")
  
    public String getJobID() {
    return jobID;
  }

  public void setJobID(String jobID) {
    this.jobID = jobID;
  }

  public ModelJobLocalEvent shardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
    return this;
  }

  /**
   * Get shardIndex
   * @return shardIndex
   **/
  @Schema(description = "")
  
    public Integer getShardIndex() {
    return shardIndex;
  }

  public void setShardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
  }

  public ModelJobLocalEvent targetNodeID(String targetNodeID) {
    this.targetNodeID = targetNodeID;
    return this;
  }

  /**
   * Get targetNodeID
   * @return targetNodeID
   **/
  @Schema(description = "")
  
    public String getTargetNodeID() {
    return targetNodeID;
  }

  public void setTargetNodeID(String targetNodeID) {
    this.targetNodeID = targetNodeID;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobLocalEvent modelJobLocalEvent = (ModelJobLocalEvent) o;
    return Objects.equals(this.eventName, modelJobLocalEvent.eventName) &&
        Objects.equals(this.jobID, modelJobLocalEvent.jobID) &&
        Objects.equals(this.shardIndex, modelJobLocalEvent.shardIndex) &&
        Objects.equals(this.targetNodeID, modelJobLocalEvent.targetNodeID);
  }

  @Override
  public int hashCode() {
    return Objects.hash(eventName, jobID, shardIndex, targetNodeID);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobLocalEvent {\n");
    
    sb.append("    eventName: ").append(toIndentedString(eventName)).append("\n");
    sb.append("    jobID: ").append(toIndentedString(jobID)).append("\n");
    sb.append("    shardIndex: ").append(toIndentedString(shardIndex)).append("\n");
    sb.append("    targetNodeID: ").append(toIndentedString(targetNodeID)).append("\n");
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
