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
 * ModelJobSpecLanguage
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobSpecLanguage   {
  @JsonProperty("Command")
  private String command = null;

  @JsonProperty("DeterministicExecution")
  private Boolean deterministicExecution = null;

  @JsonProperty("JobContext")
  private ModelStorageSpec jobContext = null;

  @JsonProperty("Language")
  private String language = null;

  @JsonProperty("LanguageVersion")
  private String languageVersion = null;

  @JsonProperty("ProgramPath")
  private String programPath = null;

  @JsonProperty("RequirementsPath")
  private String requirementsPath = null;

  public ModelJobSpecLanguage command(String command) {
    this.command = command;
    return this;
  }

  /**
   * optional program specified on commandline, like python -c \"print(1+1)\"
   * @return command
   **/
  @Schema(description = "optional program specified on commandline, like python -c \"print(1+1)\"")
  
    public String getCommand() {
    return command;
  }

  public void setCommand(String command) {
    this.command = command;
  }

  public ModelJobSpecLanguage deterministicExecution(Boolean deterministicExecution) {
    this.deterministicExecution = deterministicExecution;
    return this;
  }

  /**
   * must this job be run in a deterministic context?
   * @return deterministicExecution
   **/
  @Schema(description = "must this job be run in a deterministic context?")
  
    public Boolean isDeterministicExecution() {
    return deterministicExecution;
  }

  public void setDeterministicExecution(Boolean deterministicExecution) {
    this.deterministicExecution = deterministicExecution;
  }

  public ModelJobSpecLanguage jobContext(ModelStorageSpec jobContext) {
    this.jobContext = jobContext;
    return this;
  }

  /**
   * Get jobContext
   * @return jobContext
   **/
  @Schema(description = "")
  
    @Valid
    public ModelStorageSpec getJobContext() {
    return jobContext;
  }

  public void setJobContext(ModelStorageSpec jobContext) {
    this.jobContext = jobContext;
  }

  public ModelJobSpecLanguage language(String language) {
    this.language = language;
    return this;
  }

  /**
   * e.g. python
   * @return language
   **/
  @Schema(description = "e.g. python")
  
    public String getLanguage() {
    return language;
  }

  public void setLanguage(String language) {
    this.language = language;
  }

  public ModelJobSpecLanguage languageVersion(String languageVersion) {
    this.languageVersion = languageVersion;
    return this;
  }

  /**
   * e.g. 3.8
   * @return languageVersion
   **/
  @Schema(description = "e.g. 3.8")
  
    public String getLanguageVersion() {
    return languageVersion;
  }

  public void setLanguageVersion(String languageVersion) {
    this.languageVersion = languageVersion;
  }

  public ModelJobSpecLanguage programPath(String programPath) {
    this.programPath = programPath;
    return this;
  }

  /**
   * optional program path relative to the context dir. one of Command or ProgramPath must be specified
   * @return programPath
   **/
  @Schema(description = "optional program path relative to the context dir. one of Command or ProgramPath must be specified")
  
    public String getProgramPath() {
    return programPath;
  }

  public void setProgramPath(String programPath) {
    this.programPath = programPath;
  }

  public ModelJobSpecLanguage requirementsPath(String requirementsPath) {
    this.requirementsPath = requirementsPath;
    return this;
  }

  /**
   * optional requirements.txt (or equivalent) path relative to the context dir
   * @return requirementsPath
   **/
  @Schema(description = "optional requirements.txt (or equivalent) path relative to the context dir")
  
    public String getRequirementsPath() {
    return requirementsPath;
  }

  public void setRequirementsPath(String requirementsPath) {
    this.requirementsPath = requirementsPath;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobSpecLanguage modelJobSpecLanguage = (ModelJobSpecLanguage) o;
    return Objects.equals(this.command, modelJobSpecLanguage.command) &&
        Objects.equals(this.deterministicExecution, modelJobSpecLanguage.deterministicExecution) &&
        Objects.equals(this.jobContext, modelJobSpecLanguage.jobContext) &&
        Objects.equals(this.language, modelJobSpecLanguage.language) &&
        Objects.equals(this.languageVersion, modelJobSpecLanguage.languageVersion) &&
        Objects.equals(this.programPath, modelJobSpecLanguage.programPath) &&
        Objects.equals(this.requirementsPath, modelJobSpecLanguage.requirementsPath);
  }

  @Override
  public int hashCode() {
    return Objects.hash(command, deterministicExecution, jobContext, language, languageVersion, programPath, requirementsPath);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobSpecLanguage {\n");
    
    sb.append("    command: ").append(toIndentedString(command)).append("\n");
    sb.append("    deterministicExecution: ").append(toIndentedString(deterministicExecution)).append("\n");
    sb.append("    jobContext: ").append(toIndentedString(jobContext)).append("\n");
    sb.append("    language: ").append(toIndentedString(language)).append("\n");
    sb.append("    languageVersion: ").append(toIndentedString(languageVersion)).append("\n");
    sb.append("    programPath: ").append(toIndentedString(programPath)).append("\n");
    sb.append("    requirementsPath: ").append(toIndentedString(requirementsPath)).append("\n");
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
