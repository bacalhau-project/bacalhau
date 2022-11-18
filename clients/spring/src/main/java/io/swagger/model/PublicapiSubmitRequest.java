package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobCreatePayload;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiSubmitRequest
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiSubmitRequest   {
  @JsonProperty("client_public_key")
  private String clientPublicKey = null;

  @JsonProperty("data")
  private ModelJobCreatePayload data = null;

  @JsonProperty("signature")
  private String signature = null;

  public PublicapiSubmitRequest clientPublicKey(String clientPublicKey) {
    this.clientPublicKey = clientPublicKey;
    return this;
  }

  /**
   * The base64-encoded public key of the client:
   * @return clientPublicKey
   **/
  @Schema(required = true, description = "The base64-encoded public key of the client:")
      @NotNull

    public String getClientPublicKey() {
    return clientPublicKey;
  }

  public void setClientPublicKey(String clientPublicKey) {
    this.clientPublicKey = clientPublicKey;
  }

  public PublicapiSubmitRequest data(ModelJobCreatePayload data) {
    this.data = data;
    return this;
  }

  /**
   * Get data
   * @return data
   **/
  @Schema(required = true, description = "")
      @NotNull

    @Valid
    public ModelJobCreatePayload getData() {
    return data;
  }

  public void setData(ModelJobCreatePayload data) {
    this.data = data;
  }

  public PublicapiSubmitRequest signature(String signature) {
    this.signature = signature;
    return this;
  }

  /**
   * A base64-encoded signature of the data, signed by the client:
   * @return signature
   **/
  @Schema(required = true, description = "A base64-encoded signature of the data, signed by the client:")
      @NotNull

    public String getSignature() {
    return signature;
  }

  public void setSignature(String signature) {
    this.signature = signature;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiSubmitRequest publicapiSubmitRequest = (PublicapiSubmitRequest) o;
    return Objects.equals(this.clientPublicKey, publicapiSubmitRequest.clientPublicKey) &&
        Objects.equals(this.data, publicapiSubmitRequest.data) &&
        Objects.equals(this.signature, publicapiSubmitRequest.signature);
  }

  @Override
  public int hashCode() {
    return Objects.hash(clientPublicKey, data, signature);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiSubmitRequest {\n");
    
    sb.append("    clientPublicKey: ").append(toIndentedString(clientPublicKey)).append("\n");
    sb.append("    data: ").append(toIndentedString(data)).append("\n");
    sb.append("    signature: ").append(toIndentedString(signature)).append("\n");
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
