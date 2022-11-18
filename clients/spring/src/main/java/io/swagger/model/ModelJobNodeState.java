package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobShardState;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobNodeState
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobNodeState   {
  @JsonProperty("Shards")
  @Valid
  private Map<String, ModelJobShardState> shards = null;

  public ModelJobNodeState shards(Map<String, ModelJobShardState> shards) {
    this.shards = shards;
    return this;
  }

  public ModelJobNodeState putShardsItem(String key, ModelJobShardState shardsItem) {
    if (this.shards == null) {
      this.shards = new HashMap<String, ModelJobShardState>();
    }
    this.shards.put(key, shardsItem);
    return this;
  }

  /**
   * Get shards
   * @return shards
   **/
  @Schema(description = "")
      @Valid
    public Map<String, ModelJobShardState> getShards() {
    return shards;
  }

  public void setShards(Map<String, ModelJobShardState> shards) {
    this.shards = shards;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobNodeState modelJobNodeState = (ModelJobNodeState) o;
    return Objects.equals(this.shards, modelJobNodeState.shards);
  }

  @Override
  public int hashCode() {
    return Objects.hash(shards);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobNodeState {\n");
    
    sb.append("    shards: ").append(toIndentedString(shards)).append("\n");
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
