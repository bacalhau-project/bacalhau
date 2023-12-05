package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/bacalhau-project/bacalhau/experimental/baas/models"
	"github.com/bacalhau-project/bacalhau/experimental/baas/store"
)

func NewAPI(strg *store.Store) (*API, error) {
	return &API{store: strg}, nil
}

type API struct {
	store *store.Store
}

func (a *API) RegisterRoutes(e *echo.Echo) {
	userAPI := e.Group("/user")
	userAPI.POST("/register", a.registerUser)

	nodeAPI := e.Group("/node")
	nodeAPI.POST("/register", a.registerNode)
	nodeAPI.POST("/peers", a.findPeers)

}

type registerUserResponse struct {
	Key string
}

func (a *API) registerUser(c echo.Context) error {
	apiKey, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	newUser := models.User{}
	if a.store.DB.Create(&newUser).Error != nil {
		return err
	}

	// Create a new APIKey and associate it with the newly created user
	newAPIKey := models.APIKey{
		Key:    apiKey.String(),
		UserID: newUser.ID, // Associate with the new user's ID
	}
	if a.store.DB.Create(&newAPIKey).Error != nil {
		return err
	}

	return c.JSON(http.StatusOK, registerUserResponse{Key: apiKey.String()})
}

type registerNodeRequest struct {
	Key       string
	PeerID    string
	Addresses []string
}

func (a *API) registerNode(e echo.Context) error {
	var nodeMeta registerNodeRequest
	if err := e.Bind(&nodeMeta); err != nil {
		return err
	}

	if err := a.RegisterNode(nodeMeta); err != nil {
		return err
	}

	// Respond with success or further data if necessary
	return e.JSON(http.StatusOK, "Node registered successfully")
}

func (a *API) RegisterNode(nodeMeta registerNodeRequest) error {
	// Validate API key and find the associated user
	var apiKey models.APIKey
	if err := a.store.DB.Where("key = ?", nodeMeta.Key).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Handle API key not found
			return err
		}
		// Handle other errors
		return err
	}

	jsonAddrs, err := json.Marshal(nodeMeta.Addresses)
	if err != nil {
		return err
	}

	// Create or Update Node Metadata
	node := models.Node{
		PeerID:    nodeMeta.PeerID,
		Addresses: string(jsonAddrs),
		APIKeyID:  apiKey.ID,
	}

	// Check if a node with the given PeerID exists, update it if it does, or create a new one if it doesn't
	err = a.store.DB.Where(models.Node{PeerID: nodeMeta.PeerID}).Assign(node).FirstOrCreate(&node).Error
	if err != nil {
		// Handle creation/update error
		return err
	}

	return nil
}

type findPeersRequest struct {
	Key string
}

type findPeersResponse struct {
	Peers []peer
}

type peer struct {
	PeerID    string
	Addresses []string
}

func (a *API) findPeers(e echo.Context) error {
	var findPeers findPeersRequest
	if err := e.Bind(&findPeers); err != nil {
		return err
	}

	peers, err := a.FindPeers(findPeers)
	if err != nil {
		return err
	}

	// Return the response
	return e.JSON(http.StatusOK, findPeersResponse{Peers: peers})
}

func (a *API) FindPeers(find findPeersRequest) ([]peer, error) {
	// Validate API key and find the associated user
	var apiKey models.APIKey
	if err := a.store.DB.Where("key = ?", find.Key).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Handle API key not found
			return nil, err
		}
		// Handle other errors
		return nil, err
	}

	// Query for nodes associated with the API key
	var nodes []models.Node
	if err := a.store.DB.Where("api_key_id = ?", apiKey.ID).Find(&nodes).Error; err != nil {
		// Handle query error
		return nil, err
	}

	// Prepare the response
	var peers []peer
	for _, node := range nodes {
		var addresses []string
		if err := json.Unmarshal([]byte(node.Addresses), &addresses); err != nil {
			// Handle JSON unmarshaling error
			return nil, err
		}

		peers = append(peers, peer{
			PeerID:    node.PeerID,
			Addresses: addresses,
		})
	}

	return peers, nil
}
