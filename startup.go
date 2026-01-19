package main

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err != nil {
		return err
	}
	if loaded {
		return nil
	}

	return nil
}
