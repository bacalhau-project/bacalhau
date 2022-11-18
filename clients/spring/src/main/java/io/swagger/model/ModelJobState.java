package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobNodeState;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobState
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobState   {
  @JsonProperty("Nodes")
  @Valid
  private Map<String, ModelJobNodeState> nodes = null;

  public ModelJobState nodes(Map<String, ModelJobNodeState> nodes) {
    this.nodes = nodes;
    return this;
  }

  public ModelJobState putNodesItem(String key, ModelJobNodeState nodesItem) {
    if (this.nodes == null) {
      this.nodes = new HashMap<String, ModelJobNodeState>();
    }
    this.nodes.put(key, nodesItem);
    return this;
  }

  /**
   * Get nodes
   * @return nodes
   **/
  @Schema(description = "")
      @Valid
    public Map<String, ModelJobNodeState> getNodes() {
    return nodes;
  }

  public void setNodes(Map<String, ModelJobNodeState> nodes) {
    this.nodes = nodes;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobState modelJobState = (ModelJobState) o;
    return Objects.equals(this.nodes, modelJobState.nodes);
  }

  @Override
  public int hashCode() {
    return Objects.hash(nodes);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobState {\n");
    
    sb.append("    nodes: ").append(toIndentedString(nodes)).append("\n");
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
