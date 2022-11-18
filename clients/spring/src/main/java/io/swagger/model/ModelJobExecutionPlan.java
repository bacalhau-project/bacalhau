package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobExecutionPlan
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobExecutionPlan   {
  @JsonProperty("ShardsTotal")
  private Integer shardsTotal = null;

  public ModelJobExecutionPlan shardsTotal(Integer shardsTotal) {
    this.shardsTotal = shardsTotal;
    return this;
  }

  /**
   * how many shards are there in total for this job we are expecting this number x concurrency total JobShardState objects for this job
   * @return shardsTotal
   **/
  @Schema(description = "how many shards are there in total for this job we are expecting this number x concurrency total JobShardState objects for this job")
  
    public Integer getShardsTotal() {
    return shardsTotal;
  }

  public void setShardsTotal(Integer shardsTotal) {
    this.shardsTotal = shardsTotal;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobExecutionPlan modelJobExecutionPlan = (ModelJobExecutionPlan) o;
    return Objects.equals(this.shardsTotal, modelJobExecutionPlan.shardsTotal);
  }

  @Override
  public int hashCode() {
    return Objects.hash(shardsTotal);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobExecutionPlan {\n");
    
    sb.append("    shardsTotal: ").append(toIndentedString(shardsTotal)).append("\n");
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
