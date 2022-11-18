package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelStorageSpec;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelPublishedResult
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelPublishedResult   {
  @JsonProperty("Data")
  private ModelStorageSpec data = null;

  @JsonProperty("NodeID")
  private String nodeID = null;

  @JsonProperty("ShardIndex")
  private Integer shardIndex = null;

  public ModelPublishedResult data(ModelStorageSpec data) {
    this.data = data;
    return this;
  }

  /**
   * Get data
   * @return data
   **/
  @Schema(description = "")
  
    @Valid
    public ModelStorageSpec getData() {
    return data;
  }

  public void setData(ModelStorageSpec data) {
    this.data = data;
  }

  public ModelPublishedResult nodeID(String nodeID) {
    this.nodeID = nodeID;
    return this;
  }

  /**
   * Get nodeID
   * @return nodeID
   **/
  @Schema(description = "")
  
    public String getNodeID() {
    return nodeID;
  }

  public void setNodeID(String nodeID) {
    this.nodeID = nodeID;
  }

  public ModelPublishedResult shardIndex(Integer shardIndex) {
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


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelPublishedResult modelPublishedResult = (ModelPublishedResult) o;
    return Objects.equals(this.data, modelPublishedResult.data) &&
        Objects.equals(this.nodeID, modelPublishedResult.nodeID) &&
        Objects.equals(this.shardIndex, modelPublishedResult.shardIndex);
  }

  @Override
  public int hashCode() {
    return Objects.hash(data, nodeID, shardIndex);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelPublishedResult {\n");
    
    sb.append("    data: ").append(toIndentedString(data)).append("\n");
    sb.append("    nodeID: ").append(toIndentedString(nodeID)).append("\n");
    sb.append("    shardIndex: ").append(toIndentedString(shardIndex)).append("\n");
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
