package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelRunCommandResult
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelRunCommandResult   {
  @JsonProperty("exitCode")
  private Integer exitCode = null;

  @JsonProperty("runnerError")
  private String runnerError = null;

  @JsonProperty("stderr")
  private String stderr = null;

  @JsonProperty("stderrtruncated")
  private Boolean stderrtruncated = null;

  @JsonProperty("stdout")
  private String stdout = null;

  @JsonProperty("stdouttruncated")
  private Boolean stdouttruncated = null;

  public ModelRunCommandResult exitCode(Integer exitCode) {
    this.exitCode = exitCode;
    return this;
  }

  /**
   * exit code of the run.
   * @return exitCode
   **/
  @Schema(description = "exit code of the run.")
  
    public Integer getExitCode() {
    return exitCode;
  }

  public void setExitCode(Integer exitCode) {
    this.exitCode = exitCode;
  }

  public ModelRunCommandResult runnerError(String runnerError) {
    this.runnerError = runnerError;
    return this;
  }

  /**
   * Runner error
   * @return runnerError
   **/
  @Schema(description = "Runner error")
  
    public String getRunnerError() {
    return runnerError;
  }

  public void setRunnerError(String runnerError) {
    this.runnerError = runnerError;
  }

  public ModelRunCommandResult stderr(String stderr) {
    this.stderr = stderr;
    return this;
  }

  /**
   * stderr of the run.
   * @return stderr
   **/
  @Schema(description = "stderr of the run.")
  
    public String getStderr() {
    return stderr;
  }

  public void setStderr(String stderr) {
    this.stderr = stderr;
  }

  public ModelRunCommandResult stderrtruncated(Boolean stderrtruncated) {
    this.stderrtruncated = stderrtruncated;
    return this;
  }

  /**
   * bool describing if stderr was truncated
   * @return stderrtruncated
   **/
  @Schema(description = "bool describing if stderr was truncated")
  
    public Boolean isStderrtruncated() {
    return stderrtruncated;
  }

  public void setStderrtruncated(Boolean stderrtruncated) {
    this.stderrtruncated = stderrtruncated;
  }

  public ModelRunCommandResult stdout(String stdout) {
    this.stdout = stdout;
    return this;
  }

  /**
   * stdout of the run. Yaml provided for `describe` output
   * @return stdout
   **/
  @Schema(description = "stdout of the run. Yaml provided for `describe` output")
  
    public String getStdout() {
    return stdout;
  }

  public void setStdout(String stdout) {
    this.stdout = stdout;
  }

  public ModelRunCommandResult stdouttruncated(Boolean stdouttruncated) {
    this.stdouttruncated = stdouttruncated;
    return this;
  }

  /**
   * bool describing if stdout was truncated
   * @return stdouttruncated
   **/
  @Schema(description = "bool describing if stdout was truncated")
  
    public Boolean isStdouttruncated() {
    return stdouttruncated;
  }

  public void setStdouttruncated(Boolean stdouttruncated) {
    this.stdouttruncated = stdouttruncated;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelRunCommandResult modelRunCommandResult = (ModelRunCommandResult) o;
    return Objects.equals(this.exitCode, modelRunCommandResult.exitCode) &&
        Objects.equals(this.runnerError, modelRunCommandResult.runnerError) &&
        Objects.equals(this.stderr, modelRunCommandResult.stderr) &&
        Objects.equals(this.stderrtruncated, modelRunCommandResult.stderrtruncated) &&
        Objects.equals(this.stdout, modelRunCommandResult.stdout) &&
        Objects.equals(this.stdouttruncated, modelRunCommandResult.stdouttruncated);
  }

  @Override
  public int hashCode() {
    return Objects.hash(exitCode, runnerError, stderr, stderrtruncated, stdout, stdouttruncated);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelRunCommandResult {\n");
    
    sb.append("    exitCode: ").append(toIndentedString(exitCode)).append("\n");
    sb.append("    runnerError: ").append(toIndentedString(runnerError)).append("\n");
    sb.append("    stderr: ").append(toIndentedString(stderr)).append("\n");
    sb.append("    stderrtruncated: ").append(toIndentedString(stderrtruncated)).append("\n");
    sb.append("    stdout: ").append(toIndentedString(stdout)).append("\n");
    sb.append("    stdouttruncated: ").append(toIndentedString(stdouttruncated)).append("\n");
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
