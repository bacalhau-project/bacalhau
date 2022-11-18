package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelRunCommandResult;
import io.swagger.model.ModelStorageSpec;
import io.swagger.model.ModelVerificationResult;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelJobShardState
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelJobShardState   {
  @JsonProperty("NodeId")
  private String nodeId = null;

  @JsonProperty("PublishedResults")
  private ModelStorageSpec publishedResults = null;

  @JsonProperty("RunOutput")
  private ModelRunCommandResult runOutput = null;

  @JsonProperty("ShardIndex")
  private Integer shardIndex = null;

  @JsonProperty("State")
  private Integer state = null;

  @JsonProperty("Status")
  private String status = null;

  @JsonProperty("VerificationProposal")
  @Valid
  private List<Integer> verificationProposal = null;

  @JsonProperty("VerificationResult")
  private ModelVerificationResult verificationResult = null;

  public ModelJobShardState nodeId(String nodeId) {
    this.nodeId = nodeId;
    return this;
  }

  /**
   * which node is running this shard
   * @return nodeId
   **/
  @Schema(description = "which node is running this shard")
  
    public String getNodeId() {
    return nodeId;
  }

  public void setNodeId(String nodeId) {
    this.nodeId = nodeId;
  }

  public ModelJobShardState publishedResults(ModelStorageSpec publishedResults) {
    this.publishedResults = publishedResults;
    return this;
  }

  /**
   * Get publishedResults
   * @return publishedResults
   **/
  @Schema(description = "")
  
    @Valid
    public ModelStorageSpec getPublishedResults() {
    return publishedResults;
  }

  public void setPublishedResults(ModelStorageSpec publishedResults) {
    this.publishedResults = publishedResults;
  }

  public ModelJobShardState runOutput(ModelRunCommandResult runOutput) {
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

  public ModelJobShardState shardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
    return this;
  }

  /**
   * what shard is this we are running
   * @return shardIndex
   **/
  @Schema(description = "what shard is this we are running")
  
    public Integer getShardIndex() {
    return shardIndex;
  }

  public void setShardIndex(Integer shardIndex) {
    this.shardIndex = shardIndex;
  }

  public ModelJobShardState state(Integer state) {
    this.state = state;
    return this;
  }

  /**
   * what is the state of the shard on this node
   * @return state
   **/
  @Schema(description = "what is the state of the shard on this node")
  
    public Integer getState() {
    return state;
  }

  public void setState(Integer state) {
    this.state = state;
  }

  public ModelJobShardState status(String status) {
    this.status = status;
    return this;
  }

  /**
   * an arbitrary status message
   * @return status
   **/
  @Schema(description = "an arbitrary status message")
  
    public String getStatus() {
    return status;
  }

  public void setStatus(String status) {
    this.status = status;
  }

  public ModelJobShardState verificationProposal(List<Integer> verificationProposal) {
    this.verificationProposal = verificationProposal;
    return this;
  }

  public ModelJobShardState addVerificationProposalItem(Integer verificationProposalItem) {
    if (this.verificationProposal == null) {
      this.verificationProposal = new ArrayList<Integer>();
    }
    this.verificationProposal.add(verificationProposalItem);
    return this;
  }

  /**
   * the proposed results for this shard this will be resolved by the verifier somehow
   * @return verificationProposal
   **/
  @Schema(description = "the proposed results for this shard this will be resolved by the verifier somehow")
  
    public List<Integer> getVerificationProposal() {
    return verificationProposal;
  }

  public void setVerificationProposal(List<Integer> verificationProposal) {
    this.verificationProposal = verificationProposal;
  }

  public ModelJobShardState verificationResult(ModelVerificationResult verificationResult) {
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
    ModelJobShardState modelJobShardState = (ModelJobShardState) o;
    return Objects.equals(this.nodeId, modelJobShardState.nodeId) &&
        Objects.equals(this.publishedResults, modelJobShardState.publishedResults) &&
        Objects.equals(this.runOutput, modelJobShardState.runOutput) &&
        Objects.equals(this.shardIndex, modelJobShardState.shardIndex) &&
        Objects.equals(this.state, modelJobShardState.state) &&
        Objects.equals(this.status, modelJobShardState.status) &&
        Objects.equals(this.verificationProposal, modelJobShardState.verificationProposal) &&
        Objects.equals(this.verificationResult, modelJobShardState.verificationResult);
  }

  @Override
  public int hashCode() {
    return Objects.hash(nodeId, publishedResults, runOutput, shardIndex, state, status, verificationProposal, verificationResult);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelJobShardState {\n");
    
    sb.append("    nodeId: ").append(toIndentedString(nodeId)).append("\n");
    sb.append("    publishedResults: ").append(toIndentedString(publishedResults)).append("\n");
    sb.append("    runOutput: ").append(toIndentedString(runOutput)).append("\n");
    sb.append("    shardIndex: ").append(toIndentedString(shardIndex)).append("\n");
    sb.append("    state: ").append(toIndentedString(state)).append("\n");
    sb.append("    status: ").append(toIndentedString(status)).append("\n");
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
