package feature_customer

type Get struct {
	PhoneNumber        string `form:"phone_number" description:"phone number of the customer" example:"6590909090"`
	CustomerMetaUserId string `form:"customer_meta_user_id" description:"Meta user id of the customer"`
}
