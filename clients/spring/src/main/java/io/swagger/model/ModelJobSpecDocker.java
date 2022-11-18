package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobSpecDocker
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobSpecDocker   {
  @JsonProperty("Entrypoint")
  @Valid
  private List<String> entrypoint = null;

  @JsonProperty("EnvironmentVariables")
  @Valid
  private List<String> environmentVariables = null;

  @JsonProperty("Image")
  private String image = null;

  @JsonProperty("WorkingDirectory")
  private String workingDirectory = null;

  public ModelJobSpecDocker entrypoint(List<String> entrypoint) {
    this.entrypoint = entrypoint;
    return this;
  }

  public ModelJobSpecDocker addEntrypointItem(String entrypointItem) {
    if (this.entrypoint == null) {
      this.entrypoint = new ArrayList<String>();
    }
    this.entrypoint.add(entrypointItem);
    return this;
  }

  /**
   * optionally override the default entrypoint
   * @return entrypoint
   **/
  @Schema(description = "optionally override the default entrypoint")
  
    public List<String> getEntrypoint() {
    return entrypoint;
  }

  public void setEntrypoint(List<String> entrypoint) {
    this.entrypoint = entrypoint;
  }

  public ModelJobSpecDocker environmentVariables(List<String> environmentVariables) {
    this.environmentVariables = environmentVariables;
    return this;
  }

  public ModelJobSpecDocker addEnvironmentVariablesItem(String environmentVariablesItem) {
    if (this.environmentVariables == null) {
      this.environmentVariables = new ArrayList<String>();
    }
    this.environmentVariables.add(environmentVariablesItem);
    return this;
  }

  /**
   * a map of env to run the container with
   * @return environmentVariables
   **/
  @Schema(description = "a map of env to run the container with")
  
    public List<String> getEnvironmentVariables() {
    return environmentVariables;
  }

  public void setEnvironmentVariables(List<String> environmentVariables) {
    this.environmentVariables = environmentVariables;
  }

  public ModelJobSpecDocker image(String image) {
    this.image = image;
    return this;
  }

  /**
   * this should be pullable by docker
   * @return image
   **/
  @Schema(description = "this should be pullable by docker")
  
    public String getImage() {
    return image;
  }

  public void setImage(String image) {
    this.image = image;
  }

  public ModelJobSpecDocker workingDirectory(String workingDirectory) {
    this.workingDirectory = workingDirectory;
    return this;
  }

  /**
   * working directory inside the container
   * @return workingDirectory
   **/
  @Schema(description = "working directory inside the container")
  
    public String getWorkingDirectory() {
    return workingDirectory;
  }

  public void setWorkingDirectory(String workingDirectory) {
    this.workingDirectory = workingDirectory;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobSpecDocker modelJobSpecDocker = (ModelJobSpecDocker) o;
    return Objects.equals(this.entrypoint, modelJobSpecDocker.entrypoint) &&
        Objects.equals(this.environmentVariables, modelJobSpecDocker.environmentVariables) &&
        Objects.equals(this.image, modelJobSpecDocker.image) &&
        Objects.equals(this.workingDirectory, modelJobSpecDocker.workingDirectory);
  }

  @Override
  public int hashCode() {
    return Objects.hash(entrypoint, environmentVariables, image, workingDirectory);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobSpecDocker {\n");
    
    sb.append("    entrypoint: ").append(toIndentedString(entrypoint)).append("\n");
    sb.append("    environmentVariables: ").append(toIndentedString(environmentVariables)).append("\n");
    sb.append("    image: ").append(toIndentedString(image)).append("\n");
    sb.append("    workingDirectory: ").append(toIndentedString(workingDirectory)).append("\n");
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
