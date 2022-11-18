package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ComputenodeActiveJob;
import io.swagger.model.ModelResourceUsageData;
import io.swagger.model.RequesternodeActiveJob;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiDebugResponse
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiDebugResponse   {
  @JsonProperty("AvailableComputeCapacity")
  private ModelResourceUsageData availableComputeCapacity = null;

  @JsonProperty("ComputeJobs")
  @Valid
  private List<ComputenodeActiveJob> computeJobs = null;

  @JsonProperty("RequesterJobs")
  @Valid
  private List<RequesternodeActiveJob> requesterJobs = null;

  public PublicapiDebugResponse availableComputeCapacity(ModelResourceUsageData availableComputeCapacity) {
    this.availableComputeCapacity = availableComputeCapacity;
    return this;
  }

  /**
   * Get availableComputeCapacity
   * @return availableComputeCapacity
   **/
  @Schema(description = "")
  
    @Valid
    public ModelResourceUsageData getAvailableComputeCapacity() {
    return availableComputeCapacity;
  }

  public void setAvailableComputeCapacity(ModelResourceUsageData availableComputeCapacity) {
    this.availableComputeCapacity = availableComputeCapacity;
  }

  public PublicapiDebugResponse computeJobs(List<ComputenodeActiveJob> computeJobs) {
    this.computeJobs = computeJobs;
    return this;
  }

  public PublicapiDebugResponse addComputeJobsItem(ComputenodeActiveJob computeJobsItem) {
    if (this.computeJobs == null) {
      this.computeJobs = new ArrayList<ComputenodeActiveJob>();
    }
    this.computeJobs.add(computeJobsItem);
    return this;
  }

  /**
   * Get computeJobs
   * @return computeJobs
   **/
  @Schema(description = "")
      @Valid
    public List<ComputenodeActiveJob> getComputeJobs() {
    return computeJobs;
  }

  public void setComputeJobs(List<ComputenodeActiveJob> computeJobs) {
    this.computeJobs = computeJobs;
  }

  public PublicapiDebugResponse requesterJobs(List<RequesternodeActiveJob> requesterJobs) {
    this.requesterJobs = requesterJobs;
    return this;
  }

  public PublicapiDebugResponse addRequesterJobsItem(RequesternodeActiveJob requesterJobsItem) {
    if (this.requesterJobs == null) {
      this.requesterJobs = new ArrayList<RequesternodeActiveJob>();
    }
    this.requesterJobs.add(requesterJobsItem);
    return this;
  }

  /**
   * Get requesterJobs
   * @return requesterJobs
   **/
  @Schema(description = "")
      @Valid
    public List<RequesternodeActiveJob> getRequesterJobs() {
    return requesterJobs;
  }

  public void setRequesterJobs(List<RequesternodeActiveJob> requesterJobs) {
    this.requesterJobs = requesterJobs;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiDebugResponse publicapiDebugResponse = (PublicapiDebugResponse) o;
    return Objects.equals(this.availableComputeCapacity, publicapiDebugResponse.availableComputeCapacity) &&
        Objects.equals(this.computeJobs, publicapiDebugResponse.computeJobs) &&
        Objects.equals(this.requesterJobs, publicapiDebugResponse.requesterJobs);
  }

  @Override
  public int hashCode() {
    return Objects.hash(availableComputeCapacity, computeJobs, requesterJobs);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiDebugResponse {\n");
    
    sb.append("    availableComputeCapacity: ").append(toIndentedString(availableComputeCapacity)).append("\n");
    sb.append("    computeJobs: ").append(toIndentedString(computeJobs)).append("\n");
    sb.append("    requesterJobs: ").append(toIndentedString(requesterJobs)).append("\n");
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
