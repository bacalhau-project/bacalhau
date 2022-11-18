package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelResourceUsageConfig
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelResourceUsageConfig   {
  @JsonProperty("CPU")
  private String CPU = null;

  @JsonProperty("Disk")
  private String disk = null;

  @JsonProperty("GPU")
  private String GPU = null;

  @JsonProperty("Memory")
  private String memory = null;

  public ModelResourceUsageConfig CPU(String CPU) {
    this.CPU = CPU;
    return this;
  }

  /**
   * https://github.com/BTBurke/k8sresource string
   * @return CPU
   **/
  @Schema(description = "https://github.com/BTBurke/k8sresource string")
  
    public String getCPU() {
    return CPU;
  }

  public void setCPU(String CPU) {
    this.CPU = CPU;
  }

  public ModelResourceUsageConfig disk(String disk) {
    this.disk = disk;
    return this;
  }

  /**
   * Get disk
   * @return disk
   **/
  @Schema(description = "")
  
    public String getDisk() {
    return disk;
  }

  public void setDisk(String disk) {
    this.disk = disk;
  }

  public ModelResourceUsageConfig GPU(String GPU) {
    this.GPU = GPU;
    return this;
  }

  /**
   * unsigned integer string
   * @return GPU
   **/
  @Schema(description = "unsigned integer string")
  
    public String getGPU() {
    return GPU;
  }

  public void setGPU(String GPU) {
    this.GPU = GPU;
  }

  public ModelResourceUsageConfig memory(String memory) {
    this.memory = memory;
    return this;
  }

  /**
   * github.com/c2h5oh/datasize string
   * @return memory
   **/
  @Schema(description = "github.com/c2h5oh/datasize string")
  
    public String getMemory() {
    return memory;
  }

  public void setMemory(String memory) {
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
    ModelResourceUsageConfig modelResourceUsageConfig = (ModelResourceUsageConfig) o;
    return Objects.equals(this.CPU, modelResourceUsageConfig.CPU) &&
        Objects.equals(this.disk, modelResourceUsageConfig.disk) &&
        Objects.equals(this.GPU, modelResourceUsageConfig.GPU) &&
        Objects.equals(this.memory, modelResourceUsageConfig.memory);
  }

  @Override
  public int hashCode() {
    return Objects.hash(CPU, disk, GPU, memory);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelResourceUsageConfig {\n");
    
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
