package other

type Mailer interface {
	SendNamespaceCreated(user User, label string, tariff Tariff) error
	SendNamespaceDeleted(user User, label string) error
}

func (ml Mailer) SendNamespaceCreated(user User, label string, tariff Tariff) error {
}

func (ml Mailer) SendNamespaceDeleted(user User, label string) error {
}
