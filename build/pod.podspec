Pod::Spec.new do |spec|
  spec.name         = 'Ggdtu'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/c88032111/go-gdtu'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Gdtu Client'
  spec.source       = { :git => 'https://github.com/c88032111/go-gdtu.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Ggdtu.framework'

	spec.prepare_command = <<-CMD
    curl https://ggdtustore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Ggdtu.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
