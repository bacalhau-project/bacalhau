package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelBuildVersionInfo;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiVersionResponse
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiVersionResponse   {
  @JsonProperty("build_version_info")
  private ModelBuildVersionInfo buildVersionInfo = null;

  public PublicapiVersionResponse buildVersionInfo(ModelBuildVersionInfo buildVersionInfo) {
    this.buildVersionInfo = buildVersionInfo;
    return this;
  }

  /**
   * Get buildVersionInfo
   * @return buildVersionInfo
   **/
  @Schema(description = "")
  
    @Valid
    public ModelBuildVersionInfo getBuildVersionInfo() {
    return buildVersionInfo;
  }

  public void setBuildVersionInfo(ModelBuildVersionInfo buildVersionInfo) {
    this.buildVersionInfo = buildVersionInfo;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiVersionResponse publicapiVersionResponse = (PublicapiVersionResponse) o;
    return Objects.equals(this.buildVersionInfo, publicapiVersionResponse.buildVersionInfo);
  }

  @Override
  public int hashCode() {
    return Objects.hash(buildVersionInfo);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiVersionResponse {\n");
    
    sb.append("    buildVersionInfo: ").append(toIndentedString(buildVersionInfo)).append("\n");
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
