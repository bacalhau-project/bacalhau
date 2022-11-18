package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobShardingConfig
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobShardingConfig   {
  @JsonProperty("BatchSize")
  private Integer batchSize = null;

  @JsonProperty("GlobPattern")
  private String globPattern = null;

  @JsonProperty("GlobPatternBasePath")
  private String globPatternBasePath = null;

  public ModelJobShardingConfig batchSize(Integer batchSize) {
    this.batchSize = batchSize;
    return this;
  }

  /**
   * how many \"items\" are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes
   * @return batchSize
   **/
  @Schema(description = "how many \"items\" are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes")
  
    public Integer getBatchSize() {
    return batchSize;
  }

  public void setBatchSize(Integer batchSize) {
    this.batchSize = batchSize;
  }

  public ModelJobShardingConfig globPattern(String globPattern) {
    this.globPattern = globPattern;
    return this;
  }

  /**
   * divide the inputs up into the smallest possible unit for example /_* would mean \"all top level files or folders\" this being an empty string means \"no sharding\"
   * @return globPattern
   **/
  @Schema(description = "divide the inputs up into the smallest possible unit for example /_* would mean \"all top level files or folders\" this being an empty string means \"no sharding\"")
  
    public String getGlobPattern() {
    return globPattern;
  }

  public void setGlobPattern(String globPattern) {
    this.globPattern = globPattern;
  }

  public ModelJobShardingConfig globPatternBasePath(String globPatternBasePath) {
    this.globPatternBasePath = globPatternBasePath;
    return this;
  }

  /**
   * when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to
   * @return globPatternBasePath
   **/
  @Schema(description = "when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to")
  
    public String getGlobPatternBasePath() {
    return globPatternBasePath;
  }

  public void setGlobPatternBasePath(String globPatternBasePath) {
    this.globPatternBasePath = globPatternBasePath;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobShardingConfig modelJobShardingConfig = (ModelJobShardingConfig) o;
    return Objects.equals(this.batchSize, modelJobShardingConfig.batchSize) &&
        Objects.equals(this.globPattern, modelJobShardingConfig.globPattern) &&
        Objects.equals(this.globPatternBasePath, modelJobShardingConfig.globPatternBasePath);
  }

  @Override
  public int hashCode() {
    return Objects.hash(batchSize, globPattern, globPatternBasePath);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobShardingConfig {\n");
    
    sb.append("    batchSize: ").append(toIndentedString(batchSize)).append("\n");
    sb.append("    globPattern: ").append(toIndentedString(globPattern)).append("\n");
    sb.append("    globPatternBasePath: ").append(toIndentedString(globPatternBasePath)).append("\n");
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
