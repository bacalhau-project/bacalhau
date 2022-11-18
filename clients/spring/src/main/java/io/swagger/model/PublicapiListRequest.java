package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiListRequest
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiListRequest   {
  @JsonProperty("client_id")
  private String clientId = null;

  @JsonProperty("id")
  private String id = null;

  @JsonProperty("max_jobs")
  private Integer maxJobs = null;

  @JsonProperty("return_all")
  private Boolean returnAll = null;

  @JsonProperty("sort_by")
  private String sortBy = null;

  @JsonProperty("sort_reverse")
  private Boolean sortReverse = null;

  public PublicapiListRequest clientId(String clientId) {
    this.clientId = clientId;
    return this;
  }

  /**
   * Get clientId
   * @return clientId
   **/
  @Schema(example = "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51", description = "")
  
    public String getClientId() {
    return clientId;
  }

  public void setClientId(String clientId) {
    this.clientId = clientId;
  }

  public PublicapiListRequest id(String id) {
    this.id = id;
    return this;
  }

  /**
   * Get id
   * @return id
   **/
  @Schema(example = "9304c616-291f-41ad-b862-54e133c0149e", description = "")
  
    public String getId() {
    return id;
  }

  public void setId(String id) {
    this.id = id;
  }

  public PublicapiListRequest maxJobs(Integer maxJobs) {
    this.maxJobs = maxJobs;
    return this;
  }

  /**
   * Get maxJobs
   * @return maxJobs
   **/
  @Schema(example = "10", description = "")
  
    public Integer getMaxJobs() {
    return maxJobs;
  }

  public void setMaxJobs(Integer maxJobs) {
    this.maxJobs = maxJobs;
  }

  public PublicapiListRequest returnAll(Boolean returnAll) {
    this.returnAll = returnAll;
    return this;
  }

  /**
   * Get returnAll
   * @return returnAll
   **/
  @Schema(description = "")
  
    public Boolean isReturnAll() {
    return returnAll;
  }

  public void setReturnAll(Boolean returnAll) {
    this.returnAll = returnAll;
  }

  public PublicapiListRequest sortBy(String sortBy) {
    this.sortBy = sortBy;
    return this;
  }

  /**
   * Get sortBy
   * @return sortBy
   **/
  @Schema(example = "created_at", description = "")
  
    public String getSortBy() {
    return sortBy;
  }

  public void setSortBy(String sortBy) {
    this.sortBy = sortBy;
  }

  public PublicapiListRequest sortReverse(Boolean sortReverse) {
    this.sortReverse = sortReverse;
    return this;
  }

  /**
   * Get sortReverse
   * @return sortReverse
   **/
  @Schema(description = "")
  
    public Boolean isSortReverse() {
    return sortReverse;
  }

  public void setSortReverse(Boolean sortReverse) {
    this.sortReverse = sortReverse;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiListRequest publicapiListRequest = (PublicapiListRequest) o;
    return Objects.equals(this.clientId, publicapiListRequest.clientId) &&
        Objects.equals(this.id, publicapiListRequest.id) &&
        Objects.equals(this.maxJobs, publicapiListRequest.maxJobs) &&
        Objects.equals(this.returnAll, publicapiListRequest.returnAll) &&
        Objects.equals(this.sortBy, publicapiListRequest.sortBy) &&
        Objects.equals(this.sortReverse, publicapiListRequest.sortReverse);
  }

  @Override
  public int hashCode() {
    return Objects.hash(clientId, id, maxJobs, returnAll, sortBy, sortReverse);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiListRequest {\n");
    
    sb.append("    clientId: ").append(toIndentedString(clientId)).append("\n");
    sb.append("    id: ").append(toIndentedString(id)).append("\n");
    sb.append("    maxJobs: ").append(toIndentedString(maxJobs)).append("\n");
    sb.append("    returnAll: ").append(toIndentedString(returnAll)).append("\n");
    sb.append("    sortBy: ").append(toIndentedString(sortBy)).append("\n");
    sb.append("    sortReverse: ").append(toIndentedString(sortReverse)).append("\n");
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
