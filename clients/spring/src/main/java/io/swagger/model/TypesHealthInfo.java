package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.TypesFreeSpace;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * TypesHealthInfo
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class TypesHealthInfo   {
  @JsonProperty("FreeSpace")
  private TypesFreeSpace freeSpace = null;

  public TypesHealthInfo freeSpace(TypesFreeSpace freeSpace) {
    this.freeSpace = freeSpace;
    return this;
  }

  /**
   * Get freeSpace
   * @return freeSpace
   **/
  @Schema(description = "")
  
    @Valid
    public TypesFreeSpace getFreeSpace() {
    return freeSpace;
  }

  public void setFreeSpace(TypesFreeSpace freeSpace) {
    this.freeSpace = freeSpace;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    TypesHealthInfo typesHealthInfo = (TypesHealthInfo) o;
    return Objects.equals(this.freeSpace, typesHealthInfo.freeSpace);
  }

  @Override
  public int hashCode() {
    return Objects.hash(freeSpace);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class TypesHealthInfo {\n");
    
    sb.append("    freeSpace: ").append(toIndentedString(freeSpace)).append("\n");
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
