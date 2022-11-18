package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.TypesMountStatus;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * TypesFreeSpace
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class TypesFreeSpace   {
  @JsonProperty("IPFSMount")
  private TypesMountStatus ipFSMount = null;

  @JsonProperty("root")
  private TypesMountStatus root = null;

  @JsonProperty("tmp")
  private TypesMountStatus tmp = null;

  public TypesFreeSpace ipFSMount(TypesMountStatus ipFSMount) {
    this.ipFSMount = ipFSMount;
    return this;
  }

  /**
   * Get ipFSMount
   * @return ipFSMount
   **/
  @Schema(description = "")
  
    @Valid
    public TypesMountStatus getIpFSMount() {
    return ipFSMount;
  }

  public void setIpFSMount(TypesMountStatus ipFSMount) {
    this.ipFSMount = ipFSMount;
  }

  public TypesFreeSpace root(TypesMountStatus root) {
    this.root = root;
    return this;
  }

  /**
   * Get root
   * @return root
   **/
  @Schema(description = "")
  
    @Valid
    public TypesMountStatus getRoot() {
    return root;
  }

  public void setRoot(TypesMountStatus root) {
    this.root = root;
  }

  public TypesFreeSpace tmp(TypesMountStatus tmp) {
    this.tmp = tmp;
    return this;
  }

  /**
   * Get tmp
   * @return tmp
   **/
  @Schema(description = "")
  
    @Valid
    public TypesMountStatus getTmp() {
    return tmp;
  }

  public void setTmp(TypesMountStatus tmp) {
    this.tmp = tmp;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    TypesFreeSpace typesFreeSpace = (TypesFreeSpace) o;
    return Objects.equals(this.ipFSMount, typesFreeSpace.ipFSMount) &&
        Objects.equals(this.root, typesFreeSpace.root) &&
        Objects.equals(this.tmp, typesFreeSpace.tmp);
  }

  @Override
  public int hashCode() {
    return Objects.hash(ipFSMount, root, tmp);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class TypesFreeSpace {\n");
    
    sb.append("    ipFSMount: ").append(toIndentedString(ipFSMount)).append("\n");
    sb.append("    root: ").append(toIndentedString(root)).append("\n");
    sb.append("    tmp: ").append(toIndentedString(tmp)).append("\n");
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
