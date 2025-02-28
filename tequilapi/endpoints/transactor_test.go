/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package endpoints

import (
	"bytes"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/beneficiary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
)

var identityRegData = `{
  "beneficiary": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
  "fee": 1,
  "stake": 0
}`

func Test_RegisterIdentity(t *testing.T) {
	mockResponse := `{ "fee": 1 }`
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, &registry.FakeRegistry{RegistrationStatus: registry.Unregistered}, tr, nil, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	req, err := http.NewRequest(
		http.MethodPost,
		"/identities/{id}/register",
		bytes.NewBufferString(identityRegData),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_Get_TransactorFees(t *testing.T) {
	mockResponse := `{ "fee": 1 }`
	server := newTestTransactorServer(http.StatusOK, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, &mockSettler{
		feeToReturn: 11,
	}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/fees",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"registration":1, "settlement":1, "hermes":11, "decreaseStake":1}`, resp.Body.String())
}

func Test_SettleAsync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, &mockSettler{}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/async",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_SettleAsync_ReturnsError(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	settleRequest := `asdasdasd`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/async",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"message":"failed to unmarshal settle request: invalid character 'a' looking for beginning of value"}`, resp.Body.String())
}

func Test_SettleSync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, &mockSettler{}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/sync",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_SettleSync_ReturnsError(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiarySaver{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/sync",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"message":"settling failed: explosions everywhere"}`, resp.Body.String())
}

func Test_SettleHistory(t *testing.T) {
	t.Run("returns error on failed history retrieval", func(t *testing.T) {
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, nil, &settlementHistoryProviderMock{errToReturn: errors.New("explosions everywhere")}, &mockAddressProvider{}, &mockBeneficiarySaver{})

		req, err := http.NewRequest(http.MethodGet, "/transactor/settle/history", nil)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"message":"explosions everywhere"}`, resp.Body.String())
	})
	t.Run("returns settlement history", func(t *testing.T) {
		mockStorage := &settlementHistoryProviderMock{settlementHistoryToReturn: []pingpong.SettlementHistoryEntry{
			{
				TxHash:      common.HexToHash("0x88af51047ff2da1e3626722fe239f70c3ddd668f067b2ac8d67b280d2eff39f7"),
				Time:        time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Beneficiary: common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
				Amount:      big.NewInt(123),
				Fees:        big.NewInt(20),
			},
			{
				TxHash: common.HexToHash("0x9eea5c4da8a67929d5dd5d8b6dedb3bd44e7bd3ec299f8972f3212db8afb938a"),
				Time:   time.Date(2020, 6, 7, 8, 9, 10, 0, time.UTC),
				Amount: big.NewInt(456),
				Fees:   big.NewInt(50),
			},
		}}

		server := newTestTransactorServer(http.StatusAccepted, "")
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, nil, mockStorage, &mockAddressProvider{}, &mockBeneficiarySaver{})

		req, err := http.NewRequest(http.MethodGet, "/transactor/settle/history", nil)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(
			t,
			`{
				"items": [
					{
						"tx_hash": "0x88af51047ff2da1e3626722fe239f70c3ddd668f067b2ac8d67b280d2eff39f7",
						"provider_id": "",
						"hermes_id": "0x0000000000000000000000000000000000000000",
						"channel_address": "0x0000000000000000000000000000000000000000",
						"beneficiary":"0x4443189b9B945dD38e7bfB6167F9909451582EE5",
						"amount": 123,
						"settled_at": "2020-01-02T03:04:05Z",
						"fees": 20
					},
					{
						"tx_hash": "0x9eea5c4da8a67929d5dd5d8b6dedb3bd44e7bd3ec299f8972f3212db8afb938a",
						"provider_id": "",
						"hermes_id": "0x0000000000000000000000000000000000000000",
						"channel_address": "0x0000000000000000000000000000000000000000",
						"beneficiary": "0x0000000000000000000000000000000000000000",
						"amount": 456,
						"settled_at": "2020-06-07T08:09:10Z",
						"fees": 50
					}
				],
				"page": 1,
				"page_size": 50,
				"total_items": 2,
				"total_pages": 1
			}`,
			resp.Body.String(),
		)
	})
	t.Run("respects filters", func(t *testing.T) {
		mockStorage := &settlementHistoryProviderMock{}

		server := newTestTransactorServer(http.StatusAccepted, "")
		defer server.Close()
		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, mockIdentityRegistryInstance, tr, nil, mockStorage, &mockAddressProvider{}, &mockBeneficiarySaver{})

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history?date_from=2020-09-19&date_to=2020-09-20&provider_id=0xab1&hermes_id=0xaB2",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		expectedTimeFrom := time.Date(2020, 9, 19, 0, 0, 0, 0, time.UTC)
		expectedTimeTo := time.Date(2020, 9, 20, 23, 59, 59, 0, time.UTC)
		expectedProviderID := identity.FromAddress("0xab1")
		expectedHermesID := common.HexToAddress("0xaB2")
		assert.Equal(
			t,
			&pingpong.SettlementHistoryFilter{
				TimeFrom:   &expectedTimeFrom,
				TimeTo:     &expectedTimeTo,
				ProviderID: &expectedProviderID,
				HermesID:   &expectedHermesID,
			},
			mockStorage.calledWithFilter,
		)
	})
}

func newTestTransactorServer(mockStatus int, mockResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResponse))
	}))
}

var fakeSignerFactory = func(id identity.Identity) identity.Signer {
	return &fakeSigner{}
}

type fakeSigner struct {
}

func pad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	tmp := make([]byte, size)
	copy(tmp[size-len(b):], b)
	return tmp
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	b := make([]byte, 65)
	b = pad(b, 65)
	return identity.SignatureBytes(b), nil
}

type mockSettler struct {
	errToReturn error

	feeToReturn      uint16
	feeErrorToReturn error
}

func (ms *mockSettler) ForceSettle(_ int64, _ identity.Identity, _ common.Address) error {
	return ms.errToReturn
}

type mockBeneficiarySaver struct {
	errToReturn error
}

func (ms *mockBeneficiarySaver) SettleAndSaveBeneficiary(_ identity.Identity, _ common.Address) error {
	return ms.errToReturn
}

func (ms *mockBeneficiarySaver) BeneficiaryChangeStatus(_ identity.Identity) (*beneficiary.BeneficiaryChangeStatus, bool) {
	return &beneficiary.BeneficiaryChangeStatus{}, true
}

func (ms *mockSettler) SettleIntoStake(_ int64, providerID identity.Identity, hermesID common.Address) error {
	return nil
}

func (ms *mockSettler) GetHermesFee(_ int64, _ common.Address) (uint16, error) {
	return ms.feeToReturn, ms.feeErrorToReturn
}

type settlementHistoryProviderMock struct {
	settlementHistoryToReturn []pingpong.SettlementHistoryEntry
	errToReturn               error

	calledWithFilter *pingpong.SettlementHistoryFilter
}

func (shpm *settlementHistoryProviderMock) List(filter pingpong.SettlementHistoryFilter) ([]pingpong.SettlementHistoryEntry, error) {
	shpm.calledWithFilter = &filter
	return shpm.settlementHistoryToReturn, shpm.errToReturn
}
