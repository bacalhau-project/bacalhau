package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelDeal;
import io.swagger.model.ModelJobEvent;
import io.swagger.model.ModelJobExecutionPlan;
import io.swagger.model.ModelJobLocalEvent;
import io.swagger.model.ModelJobState;
import io.swagger.model.ModelSpec;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJob
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJob   {
  @JsonProperty("APIVersion")
  private String apIVersion = null;

  @JsonProperty("ClientID")
  private String clientID = null;

  @JsonProperty("CreatedAt")
  private String createdAt = null;

  @JsonProperty("Deal")
  private ModelDeal deal = null;

  @JsonProperty("ExecutionPlan")
  private ModelJobExecutionPlan executionPlan = null;

  @JsonProperty("ID")
  private String ID = null;

  @JsonProperty("JobEvents")
  @Valid
  private List<ModelJobEvent> jobEvents = null;

  @JsonProperty("JobState")
  private ModelJobState jobState = null;

  @JsonProperty("LocalJobEvents")
  @Valid
  private List<ModelJobLocalEvent> localJobEvents = null;

  @JsonProperty("RequesterNodeID")
  private String requesterNodeID = null;

  @JsonProperty("RequesterPublicKey")
  @Valid
  private List<Integer> requesterPublicKey = null;

  @JsonProperty("Spec")
  private ModelSpec spec = null;

  public ModelJob apIVersion(String apIVersion) {
    this.apIVersion = apIVersion;
    return this;
  }

  /**
   * Get apIVersion
   * @return apIVersion
   **/
  @Schema(example = "V1beta1", description = "")
  
    public String getApIVersion() {
    return apIVersion;
  }

  public void setApIVersion(String apIVersion) {
    this.apIVersion = apIVersion;
  }

  public ModelJob clientID(String clientID) {
    this.clientID = clientID;
    return this;
  }

  /**
   * The ID of the client that created this job.
   * @return clientID
   **/
  @Schema(example = "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51", description = "The ID of the client that created this job.")
  
    public String getClientID() {
    return clientID;
  }

  public void setClientID(String clientID) {
    this.clientID = clientID;
  }

  public ModelJob createdAt(String createdAt) {
    this.createdAt = createdAt;
    return this;
  }

  /**
   * Time the job was submitted to the bacalhau network.
   * @return createdAt
   **/
  @Schema(example = "2022-11-17T13:29:01.871140291Z", description = "Time the job was submitted to the bacalhau network.")
  
    public String getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(String createdAt) {
    this.createdAt = createdAt;
  }

  public ModelJob deal(ModelDeal deal) {
    this.deal = deal;
    return this;
  }

  /**
   * Get deal
   * @return deal
   **/
  @Schema(description = "")
  
    @Valid
    public ModelDeal getDeal() {
    return deal;
  }

  public void setDeal(ModelDeal deal) {
    this.deal = deal;
  }

  public ModelJob executionPlan(ModelJobExecutionPlan executionPlan) {
    this.executionPlan = executionPlan;
    return this;
  }

  /**
   * Get executionPlan
   * @return executionPlan
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobExecutionPlan getExecutionPlan() {
    return executionPlan;
  }

  public void setExecutionPlan(ModelJobExecutionPlan executionPlan) {
    this.executionPlan = executionPlan;
  }

  public ModelJob ID(String ID) {
    this.ID = ID;
    return this;
  }

  /**
   * The unique global ID of this job in the bacalhau network.
   * @return ID
   **/
  @Schema(example = "92d5d4ee-3765-4f78-8353-623f5f26df08", description = "The unique global ID of this job in the bacalhau network.")
  
    public String getID() {
    return ID;
  }

  public void setID(String ID) {
    this.ID = ID;
  }

  public ModelJob jobEvents(List<ModelJobEvent> jobEvents) {
    this.jobEvents = jobEvents;
    return this;
  }

  public ModelJob addJobEventsItem(ModelJobEvent jobEventsItem) {
    if (this.jobEvents == null) {
      this.jobEvents = new ArrayList<ModelJobEvent>();
    }
    this.jobEvents.add(jobEventsItem);
    return this;
  }

  /**
   * All events associated with the job
   * @return jobEvents
   **/
  @Schema(description = "All events associated with the job")
      @Valid
    public List<ModelJobEvent> getJobEvents() {
    return jobEvents;
  }

  public void setJobEvents(List<ModelJobEvent> jobEvents) {
    this.jobEvents = jobEvents;
  }

  public ModelJob jobState(ModelJobState jobState) {
    this.jobState = jobState;
    return this;
  }

  /**
   * Get jobState
   * @return jobState
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobState getJobState() {
    return jobState;
  }

  public void setJobState(ModelJobState jobState) {
    this.jobState = jobState;
  }

  public ModelJob localJobEvents(List<ModelJobLocalEvent> localJobEvents) {
    this.localJobEvents = localJobEvents;
    return this;
  }

  public ModelJob addLocalJobEventsItem(ModelJobLocalEvent localJobEventsItem) {
    if (this.localJobEvents == null) {
      this.localJobEvents = new ArrayList<ModelJobLocalEvent>();
    }
    this.localJobEvents.add(localJobEventsItem);
    return this;
  }

  /**
   * All local events associated with the job
   * @return localJobEvents
   **/
  @Schema(description = "All local events associated with the job")
      @Valid
    public List<ModelJobLocalEvent> getLocalJobEvents() {
    return localJobEvents;
  }

  public void setLocalJobEvents(List<ModelJobLocalEvent> localJobEvents) {
    this.localJobEvents = localJobEvents;
  }

  public ModelJob requesterNodeID(String requesterNodeID) {
    this.requesterNodeID = requesterNodeID;
    return this;
  }

  /**
   * The ID of the requester node that owns this job.
   * @return requesterNodeID
   **/
  @Schema(example = "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF", description = "The ID of the requester node that owns this job.")
  
    public String getRequesterNodeID() {
    return requesterNodeID;
  }

  public void setRequesterNodeID(String requesterNodeID) {
    this.requesterNodeID = requesterNodeID;
  }

  public ModelJob requesterPublicKey(List<Integer> requesterPublicKey) {
    this.requesterPublicKey = requesterPublicKey;
    return this;
  }

  public ModelJob addRequesterPublicKeyItem(Integer requesterPublicKeyItem) {
    if (this.requesterPublicKey == null) {
      this.requesterPublicKey = new ArrayList<Integer>();
    }
    this.requesterPublicKey.add(requesterPublicKeyItem);
    return this;
  }

  /**
   * The public key of the Requester node that created this job This can be used to encrypt messages back to the creator
   * @return requesterPublicKey
   **/
  @Schema(description = "The public key of the Requester node that created this job This can be used to encrypt messages back to the creator")
  
    public List<Integer> getRequesterPublicKey() {
    return requesterPublicKey;
  }

  public void setRequesterPublicKey(List<Integer> requesterPublicKey) {
    this.requesterPublicKey = requesterPublicKey;
  }

  public ModelJob spec(ModelSpec spec) {
    this.spec = spec;
    return this;
  }

  /**
   * Get spec
   * @return spec
   **/
  @Schema(description = "")
  
    @Valid
    public ModelSpec getSpec() {
    return spec;
  }

  public void setSpec(ModelSpec spec) {
    this.spec = spec;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJob modelJob = (ModelJob) o;
    return Objects.equals(this.apIVersion, modelJob.apIVersion) &&
        Objects.equals(this.clientID, modelJob.clientID) &&
        Objects.equals(this.createdAt, modelJob.createdAt) &&
        Objects.equals(this.deal, modelJob.deal) &&
        Objects.equals(this.executionPlan, modelJob.executionPlan) &&
        Objects.equals(this.ID, modelJob.ID) &&
        Objects.equals(this.jobEvents, modelJob.jobEvents) &&
        Objects.equals(this.jobState, modelJob.jobState) &&
        Objects.equals(this.localJobEvents, modelJob.localJobEvents) &&
        Objects.equals(this.requesterNodeID, modelJob.requesterNodeID) &&
        Objects.equals(this.requesterPublicKey, modelJob.requesterPublicKey) &&
        Objects.equals(this.spec, modelJob.spec);
  }

  @Override
  public int hashCode() {
    return Objects.hash(apIVersion, clientID, createdAt, deal, executionPlan, ID, jobEvents, jobState, localJobEvents, requesterNodeID, requesterPublicKey, spec);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJob {\n");
    
    sb.append("    apIVersion: ").append(toIndentedString(apIVersion)).append("\n");
    sb.append("    clientID: ").append(toIndentedString(clientID)).append("\n");
    sb.append("    createdAt: ").append(toIndentedString(createdAt)).append("\n");
    sb.append("    deal: ").append(toIndentedString(deal)).append("\n");
    sb.append("    executionPlan: ").append(toIndentedString(executionPlan)).append("\n");
    sb.append("    ID: ").append(toIndentedString(ID)).append("\n");
    sb.append("    jobEvents: ").append(toIndentedString(jobEvents)).append("\n");
    sb.append("    jobState: ").append(toIndentedString(jobState)).append("\n");
    sb.append("    localJobEvents: ").append(toIndentedString(localJobEvents)).append("\n");
    sb.append("    requesterNodeID: ").append(toIndentedString(requesterNodeID)).append("\n");
    sb.append("    requesterPublicKey: ").append(toIndentedString(requesterPublicKey)).append("\n");
    sb.append("    spec: ").append(toIndentedString(spec)).append("\n");
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
