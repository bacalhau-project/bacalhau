package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelDeal;
import io.swagger.model.ModelJobExecutionPlan;
import io.swagger.model.ModelRunCommandResult;
import io.swagger.model.ModelSpec;
import io.swagger.model.ModelStorageSpec;
import io.swagger.model.ModelVerificationResult;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobEvent
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobEvent   {
  @JsonProperty("APIVersion")
  private String apIVersion = null;

  @JsonProperty("ClientID")
  private String clientID = null;

  @JsonProperty("Deal")
  private ModelDeal deal = null;

  @JsonProperty("EventName")
  private Integer eventName = null;

  @JsonProperty("EventTime")
  private String eventTime = null;

  @JsonProperty("JobExecutionPlan")
  private ModelJobExecutionPlan jobExecutionPlan = null;

  @JsonProperty("JobID")
  private String jobID = null;

  @JsonProperty("PublishedResult")
  private ModelStorageSpec publishedResult = null;

  @JsonProperty("RunOutput")
  private ModelRunCommandResult runOutput = null;

  @JsonProperty("SenderPublicKey")
  @Valid
  private List<Integer> senderPublicKey = null;

  @JsonProperty("ShardIndex")
  private Integer shardIndex = null;

  @JsonProperty("SourceNodeID")
  private String sourceNodeID = null;

  @JsonProperty("Spec")
  private ModelSpec spec = null;

  @JsonProperty("Status")
  private String status = null;

  @JsonProperty("TargetNodeID")
  private String targetNodeID = null;

  @JsonProperty("VerificationProposal")
  @Valid
  private List<Integer> verificationProposal = null;

  @JsonProperty("VerificationResult")
  private ModelVerificationResult verificationResult = null;

  public ModelJobEvent apIVersion(String apIVersion) {
    this.apIVersion = apIVersion;
    return this;
  }

  /**
   * APIVersion of the Job
   * @return apIVersion
   **/
  @Schema(example = "V1beta1", description = "APIVersion of the Job")
  
    public String getApIVersion() {
    return apIVersion;
  }

  public void setApIVersion(String apIVersion) {
    this.apIVersion = apIVersion;
  }

  public ModelJobEvent clientID(String clientID) {
    this.clientID = clientID;
    return this;
  }

  /**
   * optional clientID if this is an externally triggered event (like create job)
   * @return clientID
   **/
  @Schema(example = "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51", description = "optional clientID if this is an externally triggered event (like create job)")
  
    public String getClientID() {
    return clientID;
  }

  public void setClientID(String clientID) {
    this.clientID = clientID;
  }

  public ModelJobEvent deal(ModelDeal deal) {
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

  public ModelJobEvent eventName(Integer eventName) {
    this.eventName = eventName;
    return this;
  }

  /**
   * Get eventName
   * @return eventName
   **/
  @Schema(description = "")
  
    public Integer getEventName() {
    return eventName;
  }

  public void setEventName(Integer eventName) {
    this.eventName = eventName;
  }

  public ModelJobEvent eventTime(String eventTime) {
    this.eventTime = eventTime;
    return this;
  }

  /**
   * Get eventTime
   * @return eventTime
   **/
  @Schema(example = "2022-11-17T13:32:55.756658941Z", description = "")
  
    public String getEventTime() {
    return eventTime;
  }

  public void setEventTime(String eventTime) {
    this.eventTime = eventTime;
  }

  public ModelJobEvent jobExecutionPlan(ModelJobExecutionPlan jobExecutionPlan) {
    this.jobExecutionPlan = jobExecutionPlan;
    return this;
  }

  /**
   * Get jobExecutionPlan
   * @return jobExecutionPlan
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobExecutionPlan getJobExecutionPlan() {
    return jobExecutionPlan;
  }

  public void setJobExecutionPlan(ModelJobExecutionPlan jobExecutionPlan) {
    this.jobExecutionPlan = jobExecutionPlan;
  }

  public ModelJobEvent jobID(String jobID) {
    this.jobID = jobID;
    return this;
  }

  /**
   * Get jobID
   * @return jobID
   **/
  @Schema(example = "9304c616-291f-41ad-b862-54e133c0149e", description = "")
  
    public String getJobID() {
    return jobID;
  }

  public void setJobID(String jobID) {
    this.jobID = jobID;
  }

  public ModelJobEvent publishedResult(ModelStorageSpec publishedResult) {
    this.publishedResult = publishedResult;
    return this;
  }

  /**
   * Get publishedResult
   * @return publishedResult
   **/
  @Schema(description = "")
  
    @Valid
    public ModelStorageSpec getPublishedResult() {
    return publishedResult;
  }

  public void setPublishedResult(ModelStorageSpec publishedResult) {
    this.publishedResult = publishedResult;
  }

  public ModelJobEvent runOutput(ModelRunCommandResult runOutput) {
    this.runOutput = runOutput;
    return this;
  }

  /**
   * Get runOutput
   * @return runOutput
   **/
  @Schema(description = "")
  
    @Valid
    public ModelRunCommandResult getRunOutput() {
    return runOutput;
  }

  public void setRunOutput(ModelRunCommandResult runOutput) {
    this.runOutput = runOutput;
  }

  public ModelJobEvent senderPublicKey(List<Integer> senderPublicKey) {
    this.senderPublicKey = senderPublicKey;
    return this;
  }

  public ModelJobEvent addSenderPublicKeyItem(Integer senderPublicKeyItem) {
    if (this.senderPublicKey == null) {
      this.senderPublicKey = new ArrayList<Integer>();
    }
    this.senderPublicKey.add(senderPublicKeyItem);
    return this;
  }

  /**
   * Get senderPublicKey
   * @return senderPublicKey
   **/
  @Schema(description = "")
  
    public List<Integer> getSenderPublicKey() {
    return senderPublicKey;
  }

  public void setSenderPublicKey(List<Integer> senderPublicKey) {
    this.senderPublicKey = senderPublicKey;
  }

  public ModelJobEvent shardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
    return this;
  }

  /**
   * what shard is this event for
   * @return shardIndex
   **/
  @Schema(description = "what shard is this event for")
  
    public Integer getShardIndex() {
    return shardIndex;
  }

  public void setShardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
  }

  public ModelJobEvent sourceNodeID(String sourceNodeID) {
    this.sourceNodeID = sourceNodeID;
    return this;
  }

  /**
   * the node that emitted this event
   * @return sourceNodeID
   **/
  @Schema(example = "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF", description = "the node that emitted this event")
  
    public String getSourceNodeID() {
    return sourceNodeID;
  }

  public void setSourceNodeID(String sourceNodeID) {
    this.sourceNodeID = sourceNodeID;
  }

  public ModelJobEvent spec(ModelSpec spec) {
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

  public ModelJobEvent status(String status) {
    this.status = status;
    return this;
  }

  /**
   * Get status
   * @return status
   **/
  @Schema(example = "Got results proposal of length: 0", description = "")
  
    public String getStatus() {
    return status;
  }

  public void setStatus(String status) {
    this.status = status;
  }

  public ModelJobEvent targetNodeID(String targetNodeID) {
    this.targetNodeID = targetNodeID;
    return this;
  }

  /**
   * the node that this event is for e.g. \"AcceptJobBid\" was emitted by Requester but it targeting compute node
   * @return targetNodeID
   **/
  @Schema(example = "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL", description = "the node that this event is for e.g. \"AcceptJobBid\" was emitted by Requester but it targeting compute node")
  
    public String getTargetNodeID() {
    return targetNodeID;
  }

  public void setTargetNodeID(String targetNodeID) {
    this.targetNodeID = targetNodeID;
  }

  public ModelJobEvent verificationProposal(List<Integer> verificationProposal) {
    this.verificationProposal = verificationProposal;
    return this;
  }

  public ModelJobEvent addVerificationProposalItem(Integer verificationProposalItem) {
    if (this.verificationProposal == null) {
      this.verificationProposal = new ArrayList<Integer>();
    }
    this.verificationProposal.add(verificationProposalItem);
    return this;
  }

  /**
   * Get verificationProposal
   * @return verificationProposal
   **/
  @Schema(description = "")
  
    public List<Integer> getVerificationProposal() {
    return verificationProposal;
  }

  public void setVerificationProposal(List<Integer> verificationProposal) {
    this.verificationProposal = verificationProposal;
  }

  public ModelJobEvent verificationResult(ModelVerificationResult verificationResult) {
    this.verificationResult = verificationResult;
    return this;
  }

  /**
   * Get verificationResult
   * @return verificationResult
   **/
  @Schema(description = "")
  
    @Valid
    public ModelVerificationResult getVerificationResult() {
    return verificationResult;
  }

  public void setVerificationResult(ModelVerificationResult verificationResult) {
    this.verificationResult = verificationResult;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelJobEvent modelJobEvent = (ModelJobEvent) o;
    return Objects.equals(this.apIVersion, modelJobEvent.apIVersion) &&
        Objects.equals(this.clientID, modelJobEvent.clientID) &&
        Objects.equals(this.deal, modelJobEvent.deal) &&
        Objects.equals(this.eventName, modelJobEvent.eventName) &&
        Objects.equals(this.eventTime, modelJobEvent.eventTime) &&
        Objects.equals(this.jobExecutionPlan, modelJobEvent.jobExecutionPlan) &&
        Objects.equals(this.jobID, modelJobEvent.jobID) &&
        Objects.equals(this.publishedResult, modelJobEvent.publishedResult) &&
        Objects.equals(this.runOutput, modelJobEvent.runOutput) &&
        Objects.equals(this.senderPublicKey, modelJobEvent.senderPublicKey) &&
        Objects.equals(this.shardIndex, modelJobEvent.shardIndex) &&
        Objects.equals(this.sourceNodeID, modelJobEvent.sourceNodeID) &&
        Objects.equals(this.spec, modelJobEvent.spec) &&
        Objects.equals(this.status, modelJobEvent.status) &&
        Objects.equals(this.targetNodeID, modelJobEvent.targetNodeID) &&
        Objects.equals(this.verificationProposal, modelJobEvent.verificationProposal) &&
        Objects.equals(this.verificationResult, modelJobEvent.verificationResult);
  }

  @Override
  public int hashCode() {
    return Objects.hash(apIVersion, clientID, deal, eventName, eventTime, jobExecutionPlan, jobID, publishedResult, runOutput, senderPublicKey, shardIndex, sourceNodeID, spec, status, targetNodeID, verificationProposal, verificationResult);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobEvent {\n");
    
    sb.append("    apIVersion: ").append(toIndentedString(apIVersion)).append("\n");
    sb.append("    clientID: ").append(toIndentedString(clientID)).append("\n");
    sb.append("    deal: ").append(toIndentedString(deal)).append("\n");
    sb.append("    eventName: ").append(toIndentedString(eventName)).append("\n");
    sb.append("    eventTime: ").append(toIndentedString(eventTime)).append("\n");
    sb.append("    jobExecutionPlan: ").append(toIndentedString(jobExecutionPlan)).append("\n");
    sb.append("    jobID: ").append(toIndentedString(jobID)).append("\n");
    sb.append("    publishedResult: ").append(toIndentedString(publishedResult)).append("\n");
    sb.append("    runOutput: ").append(toIndentedString(runOutput)).append("\n");
    sb.append("    senderPublicKey: ").append(toIndentedString(senderPublicKey)).append("\n");
    sb.append("    shardIndex: ").append(toIndentedString(shardIndex)).append("\n");
    sb.append("    sourceNodeID: ").append(toIndentedString(sourceNodeID)).append("\n");
    sb.append("    spec: ").append(toIndentedString(spec)).append("\n");
    sb.append("    status: ").append(toIndentedString(status)).append("\n");
    sb.append("    targetNodeID: ").append(toIndentedString(targetNodeID)).append("\n");
    sb.append("    verificationProposal: ").append(toIndentedString(verificationProposal)).append("\n");
    sb.append("    verificationResult: ").append(toIndentedString(verificationResult)).append("\n");
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
