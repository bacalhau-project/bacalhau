package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import java.math.BigDecimal;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelResourceUsageData
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelResourceUsageData   {
  @JsonProperty("CPU")
  private BigDecimal CPU = null;

  @JsonProperty("Disk")
  private Integer disk = null;

  @JsonProperty("GPU")
  private Integer GPU = null;

  @JsonProperty("Memory")
  private Integer memory = null;

  public ModelResourceUsageData CPU(BigDecimal CPU) {
    this.CPU = CPU;
    return this;
  }

  /**
   * cpu units
   * @return CPU
   **/
  @Schema(example = "9.600000000000001", description = "cpu units")
  
    @Valid
    public BigDecimal getCPU() {
    return CPU;
  }

  public void setCPU(BigDecimal CPU) {
    this.CPU = CPU;
  }

  public ModelResourceUsageData disk(Integer disk) {
    this.disk = disk;
    return this;
  }

  /**
   * bytes
   * @return disk
   **/
  @Schema(example = "212663867801", description = "bytes")
  
    public Integer getDisk() {
    return disk;
  }

  public void setDisk(Integer disk) {
    this.disk = disk;
  }

  public ModelResourceUsageData GPU(Integer GPU) {
    this.GPU = GPU;
    return this;
  }

  /**
   * Get GPU
   * @return GPU
   **/
  @Schema(example = "1", description = "")
  
    public Integer getGPU() {
    return GPU;
  }

  public void setGPU(Integer GPU) {
    this.GPU = GPU;
  }

  public ModelResourceUsageData memory(Integer memory) {
    this.memory = memory;
    return this;
  }

  /**
   * bytes
   * @return memory
   **/
  @Schema(example = "27487790694", description = "bytes")
  
    public Integer getMemory() {
    return memory;
  }

  public void setMemory(Integer memory) {
    this.memory = memory;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelResourceUsageData modelResourceUsageData = (ModelResourceUsageData) o;
    return Objects.equals(this.CPU, modelResourceUsageData.CPU) &&
        Objects.equals(this.disk, modelResourceUsageData.disk) &&
        Objects.equals(this.GPU, modelResourceUsageData.GPU) &&
        Objects.equals(this.memory, modelResourceUsageData.memory);
  }

  @Override
  public int hashCode() {
    return Objects.hash(CPU, disk, GPU, memory);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelResourceUsageData {\n");
    
    sb.append("    CPU: ").append(toIndentedString(CPU)).append("\n");
    sb.append("    disk: ").append(toIndentedString(disk)).append("\n");
    sb.append("    GPU: ").append(toIndentedString(GPU)).append("\n");
    sb.append("    memory: ").append(toIndentedString(memory)).append("\n");
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
