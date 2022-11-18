package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelStorageSpec;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobSpecWasm
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobSpecWasm   {
  @JsonProperty("EntryPoint")
  private String entryPoint = null;

  @JsonProperty("EnvironmentVariables")
  @Valid
  private Map<String, String> environmentVariables = null;

  @JsonProperty("ImportModules")
  @Valid
  private List<ModelStorageSpec> importModules = null;

  @JsonProperty("Parameters")
  @Valid
  private List<String> parameters = null;

  public ModelJobSpecWasm entryPoint(String entryPoint) {
    this.entryPoint = entryPoint;
    return this;
  }

  /**
   * The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be `_start`, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function.
   * @return entryPoint
   **/
  @Schema(description = "The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be `_start`, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function.")
  
    public String getEntryPoint() {
    return entryPoint;
  }

  public void setEntryPoint(String entryPoint) {
    this.entryPoint = entryPoint;
  }

  public ModelJobSpecWasm environmentVariables(Map<String, String> environmentVariables) {
    this.environmentVariables = environmentVariables;
    return this;
  }

  public ModelJobSpecWasm putEnvironmentVariablesItem(String key, String environmentVariablesItem) {
    if (this.environmentVariables == null) {
      this.environmentVariables = new HashMap<String, String>();
    }
    this.environmentVariables.put(key, environmentVariablesItem);
    return this;
  }

  /**
   * The variables available in the environment of the running program.
   * @return environmentVariables
   **/
  @Schema(description = "The variables available in the environment of the running program.")
  
    public Map<String, String> getEnvironmentVariables() {
    return environmentVariables;
  }

  public void setEnvironmentVariables(Map<String, String> environmentVariables) {
    this.environmentVariables = environmentVariables;
  }

  public ModelJobSpecWasm importModules(List<ModelStorageSpec> importModules) {
    this.importModules = importModules;
    return this;
  }

  public ModelJobSpecWasm addImportModulesItem(ModelStorageSpec importModulesItem) {
    if (this.importModules == null) {
      this.importModules = new ArrayList<ModelStorageSpec>();
    }
    this.importModules.add(importModulesItem);
    return this;
  }

  /**
   * TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule.
   * @return importModules
   **/
  @Schema(description = "TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule.")
      @Valid
    public List<ModelStorageSpec> getImportModules() {
    return importModules;
  }

  public void setImportModules(List<ModelStorageSpec> importModules) {
    this.importModules = importModules;
  }

  public ModelJobSpecWasm parameters(List<String> parameters) {
    this.parameters = parameters;
    return this;
  }

  public ModelJobSpecWasm addParametersItem(String parametersItem) {
    if (this.parameters == null) {
      this.parameters = new ArrayList<String>();
    }
    this.parameters.add(parametersItem);
    return this;
  }

  /**
   * The arguments supplied to the program (i.e. as ARGV).
   * @return parameters
   **/
  @Schema(description = "The arguments supplied to the program (i.e. as ARGV).")
  
    public List<String> getParameters() {
    return parameters;
  }

  public void setParameters(List<String> parameters) {
    this.parameters = parameters;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobSpecWasm modelJobSpecWasm = (ModelJobSpecWasm) o;
    return Objects.equals(this.entryPoint, modelJobSpecWasm.entryPoint) &&
        Objects.equals(this.environmentVariables, modelJobSpecWasm.environmentVariables) &&
        Objects.equals(this.importModules, modelJobSpecWasm.importModules) &&
        Objects.equals(this.parameters, modelJobSpecWasm.parameters);
  }

  @Override
  public int hashCode() {
    return Objects.hash(entryPoint, environmentVariables, importModules, parameters);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobSpecWasm {\n");
    
    sb.append("    entryPoint: ").append(toIndentedString(entryPoint)).append("\n");
    sb.append("    environmentVariables: ").append(toIndentedString(environmentVariables)).append("\n");
    sb.append("    importModules: ").append(toIndentedString(importModules)).append("\n");
    sb.append("    parameters: ").append(toIndentedString(parameters)).append("\n");
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
