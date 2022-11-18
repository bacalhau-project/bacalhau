package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * TypesMountStatus
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class TypesMountStatus   {
  @JsonProperty("All")
  private Integer all = null;

  @JsonProperty("Free")
  private Integer free = null;

  @JsonProperty("Used")
  private Integer used = null;

  public TypesMountStatus all(Integer all) {
    this.all = all;
    return this;
  }

  /**
   * Get all
   * @return all
   **/
  @Schema(description = "")
  
    public Integer getAll() {
    return all;
  }

  public void setAll(Integer all) {
    this.all = all;
  }

  public TypesMountStatus free(Integer free) {
    this.free = free;
    return this;
  }

  /**
   * Get free
   * @return free
   **/
  @Schema(description = "")
  
    public Integer getFree() {
    return free;
  }

  public void setFree(Integer free) {
    this.free = free;
  }

  public TypesMountStatus used(Integer used) {
    this.used = used;
    return this;
  }

  /**
   * Get used
   * @return used
   **/
  @Schema(description = "")
  
    public Integer getUsed() {
    return used;
  }

  public void setUsed(Integer used) {
    this.used = used;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    TypesMountStatus typesMountStatus = (TypesMountStatus) o;
    return Objects.equals(this.all, typesMountStatus.all) &&
        Objects.equals(this.free, typesMountStatus.free) &&
        Objects.equals(this.used, typesMountStatus.used);
  }

  @Override
  public int hashCode() {
    return Objects.hash(all, free, used);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class TypesMountStatus {\n");
    
    sb.append("    all: ").append(toIndentedString(all)).append("\n");
    sb.append("    free: ").append(toIndentedString(free)).append("\n");
    sb.append("    used: ").append(toIndentedString(used)).append("\n");
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
