package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelResourceUsageData;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ComputenodeActiveJob
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ComputenodeActiveJob   {
  @JsonProperty("CapacityRequirements")
  private ModelResourceUsageData capacityRequirements = null;

  @JsonProperty("ShardID")
  private String shardID = null;

  @JsonProperty("State")
  private String state = null;

  public ComputenodeActiveJob capacityRequirements(ModelResourceUsageData capacityRequirements) {
    this.capacityRequirements = capacityRequirements;
    return this;
  }

  /**
   * Get capacityRequirements
   * @return capacityRequirements
   **/
  @Schema(description = "")
  
    @Valid
    public ModelResourceUsageData getCapacityRequirements() {
    return capacityRequirements;
  }

  public void setCapacityRequirements(ModelResourceUsageData capacityRequirements) {
    this.capacityRequirements = capacityRequirements;
  }

  public ComputenodeActiveJob shardID(String shardID) {
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

  public ComputenodeActiveJob state(String state) {
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
    ComputenodeActiveJob computenodeActiveJob = (ComputenodeActiveJob) o;
    return Objects.equals(this.capacityRequirements, computenodeActiveJob.capacityRequirements) &&
        Objects.equals(this.shardID, computenodeActiveJob.shardID) &&
        Objects.equals(this.state, computenodeActiveJob.state);
  }

  @Override
  public int hashCode() {
    return Objects.hash(capacityRequirements, shardID, state);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ComputenodeActiveJob {\n");
    
    sb.append("    capacityRequirements: ").append(toIndentedString(capacityRequirements)).append("\n");
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
